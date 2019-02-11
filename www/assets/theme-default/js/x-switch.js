//
function create_element(name, attrs=[], children=[]) {
    var ele = document.createElement(name);
    for (const item of attrs) ele.setAttribute(item[0], item[1]);
    for (const item of children) ele.appendChild(item);
    return ele;
}
function dcTN(string) {
    return document.createTextNode(string);
}

customElements.define("x-switch", class extends HTMLElement {
    constructor() {
        super();
        const { mode } = this.dataset;
        this.appendChild(create_element("label", [["class","switch-o"]], [
            create_element("input", [["type","checkbox"]]),
            create_element("span"),
        ]));
        this.firstElementChild.firstElementChild.addEventListener("change", (e) => {
            if (e.target.checked) {
                document.body.classList.add(mode);
                localStorage.setItem(`mode_${mode}`, "true");
            }
            else {
                document.body.classList.remove(mode);
                localStorage.removeItem(`mode_${mode}`);
            }
        });
        if (localStorage.getItem(`mode_${mode}`) !== null) {
            this.firstElementChild.firstElementChild.checked = true;
            document.body.classList.add(mode);
        }
    }
});

document.head.appendChild(create_element("style", [], [document.createTextNode(`
    x-switch {
        display: flex;
    }
    x-switch label {
        position: relative;
        display: inline-block;
        width: 2em;
        height: 1em;
    }
    x-switch label input {
        opacity: 0;
        width: 0;
        height: 0;
    }
    x-switch label input:checked + span::before {
        transform: translateX(1em);
    }
    x-switch label span {
        position: absolute;
        cursor: pointer;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background-color: #ccc;
        -webkit-transition: .4s;
        transition: .4s;
        border-radius: .25em;
    }
    x-switch label span::before {
        position: absolute;
        content: "";
        height: .75em;
        width: .75em;
        left: .125em;
        bottom: .125em;
        background-color: white;
        -webkit-transition: .4s;
        transition: .4s;
        border-radius: 50%;
    }
`)]));
