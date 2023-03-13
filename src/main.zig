const std = @import("std");
const string = []const u8;
const builtin = @import("builtin");
const http = @import("mango_pie");
const signal = @import("signal");
const flag = @import("flag");
const pek = @import("pek");
const files = @import("self/files");
const time = @import("time");
const mime = @import("mime");
const extras = @import("extras");

var global_running = std.atomic.Atomic(bool).init(true);
var public: ?Location = null;

comptime {
    std.debug.assert(builtin.os.tag == .linux);
    std.debug.assert(builtin.cpu.arch.ptrBitWidth() >= 64);
}

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{ .stack_trace_frames = 16 }){};
    defer std.debug.assert(!gpa.deinit());
    const alloc = if (builtin.mode == .Debug) gpa.allocator() else std.heap.c_allocator;

    //

    signal.listenFor(std.os.linux.SIG.INT, handle_sig);
    signal.listenFor(std.os.linux.SIG.TERM, handle_sig);

    //

    flag.init(alloc);
    defer flag.deinit();

    try flag.addSingle("public");
    try flag.addSingle("port");

    _ = try flag.parse(.double);
    try flag.parseEnv();

    //

    blk: {
        const p = flag.getSingle("public") orelse break :blk;
        std.debug.assert(p.len > 0);
        if (p.len > 1) std.debug.assert(!std.mem.endsWith(u8, p, "/"));

        const d = try std.fs.cwd().openIterableDir(p, .{});
        const s = try d.dir.stat();
        std.debug.assert(s.kind == .Directory);

        public = .{ p, d };
    }
    defer if (public) |_| public.?[1].close();

    //

    // Create the server socket
    const listen_port = try std.fmt.parseUnsigned(u16, flag.getSingle("port") orelse "8000", 10);
    const server_fd = try http.createSocket(listen_port);
    std.log.info("starting server on port {d}", .{listen_port});

    // Create the server
    var server: http.Server = undefined;
    try server.init(alloc, .{}, &global_running, server_fd, handleRequest);
    defer server.deinit();

    try server.run(std.time.ns_per_s);
}

fn handle_sig() void {
    std.log.info("exiting safely...", .{});
    global_running.store(false, .SeqCst);
}

fn handleRequest(per_request_allocator: std.mem.Allocator, peer: http.Peer, res_writer: http.ResponseWriter, req: http.Request) anyerror!http.Response {
    std.log.debug("IN HANDLER addr={} method: {s}, path: {s}, body: \"{?s}\"", .{ peer.addr, @tagName(req.method), req.path, req.body });

    if (public) |_| {
        if (std.mem.eql(u8, req.path, "/public/")) {
            return respondListing(per_request_allocator, peer, res_writer, req, public.?);
        }

        if (std.mem.startsWith(u8, req.path, "/public/")) {
            const rp = try std.fmt.allocPrint(per_request_allocator, "{s}{s}", .{ flag.getSingle("public").?, req.path[7..] });
            const p = if (std.mem.startsWith(u8, rp, "//")) rp[1..] else rp;

            if (std.mem.endsWith(u8, req.path, "/")) {
                var idir = try public.?[1].dir.openIterableDir(req.path[8..], .{});
                defer idir.close();
                return respondListing(per_request_allocator, peer, res_writer, req, .{ p, idir });
            }

            var file = public.?[1].dir.openFile(req.path[8..], .{}) catch |err| switch (err) {
                error.FileNotFound,
                error.AccessDenied,
                => return @unionInit(http.Response, "response", .{ .status_code = .not_found, .headers = http.Headers.from(&.{}) }),

                error.BadPathName,
                error.NameTooLong,
                error.InvalidUtf8,
                => return @unionInit(http.Response, "response", .{ .status_code = .bad_request, .headers = http.Headers.from(&.{}) }),

                else => |e| return e,
            };
            defer file.close();
            const stat = try file.stat();

            if (stat.kind == .Directory) {
                return http.Response{
                    .response = .{
                        .status_code = .found,
                        .headers = http.Headers.from(&.{
                            .{ .name = "Location", .value = try std.fmt.allocPrint(per_request_allocator, "{s}/", .{req.path}) },
                        }),
                    },
                };
            }
            return http.Response{
                .send_file = .{
                    .status_code = .ok,
                    .headers = http.Headers.from(&.{}),
                    .path = p,
                },
            };
        }
    }

    if (std.mem.eql(u8, req.path, "/")) {
        try res_writer.writeAll(files.@"/index.html");
        return http.Response{
            .response = .{
                .status_code = .ok,
                .headers = http.Headers.from(&.{
                    .{ .name = "Content-Type", .value = "text/html" },
                }),
            },
        };
    }

    inline for (comptime std.meta.declarations(files)) |decl| {
        if (std.mem.eql(u8, req.path, decl.name)) {
            try res_writer.writeAll(@field(files, decl.name));
            return http.Response{
                .response = .{
                    .status_code = .ok,
                    .headers = http.Headers.from(&.{
                        .{ .name = "Content-Type", .value = mime.typeByExtension(std.fs.path.extension(decl.name)) orelse "application/octet-stream" },
                    }),
                },
            };
        }
    }

    return @unionInit(http.Response, "response", .{ .status_code = .not_found, .headers = http.Headers.from(&.{}) });
}

fn respondListing(alloc: std.mem.Allocator, peer: http.Peer, res_writer: http.ResponseWriter, req: http.Request, loc: Location) anyerror!http.Response {
    var the_files = std.ArrayList(File).init(alloc);
    var iter: std.fs.IterableDir.Iterator = loc[1].iterate();
    while (try iter.next()) |entry| {
        const name = try alloc.dupe(u8, entry.name);
        const stat: std.fs.File.Stat = loc[1].dir.statFile(entry.name) catch |err| switch (err) {
            error.FileNotFound => continue, // this can be a broken symlink
            else => |e| return e,
        };
        const mod_raw_ms = @intCast(u64, @divTrunc(stat.mtime, std.time.ns_per_ms));
        const is_folder = entry.kind == .Directory;
        const extension: string = if (is_folder) "folder" else (nullify(std.fs.path.extension(name)) orelse @as(string, "."))[1..];

        try the_files.append(.{
            .name = name,
            .kind = entry.kind,
            .is_file = @boolToInt(!is_folder),
            .ext = extension,
            .mod_raw = mod_raw_ms,
            .mod = try time.DateTime.initUnixMs(mod_raw_ms).formatAlloc(alloc, "YYYY-MM-DD"),
            .size = stat.size,
        });
    }
    const the_files_final = try the_files.toOwnedSlice();
    std.sort.sort(File, the_files_final, {}, lessThanFile);

    const page = files.@"/listing.pek";
    const tmpl = comptime pek.parse(page);
    try pek.compile(@This(), alloc, res_writer, tmpl, .{
        .path = req.path,
        .base = @as(string, "/"),
        .version = @as(string, "0.1"),
        .user = User{
            .name = @as(string, "Guest"),
            .provider = try std.fmt.allocPrint(alloc, "{}", .{peer.addr}),
        },
        .files = the_files_final,
    });
    return http.Response{
        .response = .{
            .status_code = .ok,
            .headers = http.Headers.from(&.{
                .{ .name = "Content-Type", .value = "text/html" },
            }),
        },
    };
}

const Location = struct {
    string,
    std.fs.IterableDir,
};

pub const User = struct {
    name: string,
    provider: string,
};

pub const File = struct {
    name: string,
    kind: std.fs.File.Kind,
    is_file: u1,
    ext: string,
    mod_raw: u64,
    mod: string,
    size: u64,
};

pub fn pek_url_name(alloc: std.mem.Allocator, writer: std.ArrayList(u8).Writer, name: string) !void {
    _ = alloc;
    try writer.writeAll(name);
}

fn nullify(s: string) ?string {
    if (s.len == 0) return null;
    return s;
}

fn lessThanFile(context: void, lhs: File, rhs: File) bool {
    _ = context;
    if (lhs.is_file == rhs.is_file) return extras.lessThanSlice(string)({}, lhs.name, rhs.name);
    if (!(lhs.is_file == 1) and rhs.is_file == 1) return true;
    return false;
}
