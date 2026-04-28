var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var _HTMLElement_dataset, _HTMLElement_customElementDefineCallback, _a, _b, _c, _d, _e, _f, _g, _h, _j, _k, _l, _m, _o;
import Element from '../element/Element.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CSSStyleDeclaration from '../../css/declaration/CSSStyleDeclaration.js';
import PointerEvent from '../../event/events/PointerEvent.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import DOMException from '../../exception/DOMException.js';
import HTMLElementUtility from './HTMLElementUtility.js';
import HTMLElementNamedNodeMap from './HTMLElementNamedNodeMap.js';
import NodeList from '../node/NodeList.js';
import HTMLCollection from '../element/HTMLCollection.js';
import DatasetFactory from '../element/DatasetFactory.js';
/**
 * HTML Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement.
 */
export default class HTMLElement extends Element {
    constructor() {
        super(...arguments);
        // Events
        this.oncopy = null;
        this.oncut = null;
        this.onpaste = null;
        this.oninvalid = null;
        this.onanimationcancel = null;
        this.onanimationend = null;
        this.onanimationiteration = null;
        this.onanimationstart = null;
        this.onbeforeinput = null;
        this.oninput = null;
        this.onchange = null;
        this.ongotpointercapture = null;
        this.onlostpointercapture = null;
        this.onpointercancel = null;
        this.onpointerdown = null;
        this.onpointerenter = null;
        this.onpointerleave = null;
        this.onpointermove = null;
        this.onpointerout = null;
        this.onpointerover = null;
        this.onpointerup = null;
        this.ontransitioncancel = null;
        this.ontransitionend = null;
        this.ontransitionrun = null;
        this.ontransitionstart = null;
        // Internal properties
        this[_a] = new HTMLElementNamedNodeMap(this);
        this[_b] = '';
        this[_c] = 'inherit';
        this[_d] = false;
        this[_e] = 0;
        this[_f] = 0;
        this[_g] = 0;
        this[_h] = 0;
        this[_j] = 0;
        this[_k] = 0;
        this[_l] = 0;
        this[_m] = 0;
        this[_o] = null;
        // Private properties
        _HTMLElement_dataset.set(this, null);
        _HTMLElement_customElementDefineCallback.set(this, null);
    }
    /**
     * Returns access key.
     *
     * @returns Access key.
     */
    get accessKey() {
        return this[PropertySymbol.accessKey];
    }
    /**
     * Sets access key.
     *
     * @param accessKey Access key.
     */
    set accessKey(accessKey) {
        this[PropertySymbol.accessKey] = accessKey;
    }
    /**
     * Returns content editable.
     *
     * @returns Content editable.
     */
    get contentEditable() {
        return this[PropertySymbol.contentEditable];
    }
    /**
     * Sets content editable.
     *
     * @param contentEditable Content editable.
     */
    set contentEditable(contentEditable) {
        this[PropertySymbol.contentEditable] = contentEditable;
    }
    /**
     * Returns is content editable.
     *
     * @returns Is content editable.
     */
    get isContentEditable() {
        return this[PropertySymbol.isContentEditable];
    }
    /**
     * Returns offset height.
     *
     * @returns Offset height.
     */
    get offsetHeight() {
        return this[PropertySymbol.offsetHeight];
    }
    /**
     * Returns offset width.
     *
     * @returns Offset width.
     */
    get offsetWidth() {
        return this[PropertySymbol.offsetWidth];
    }
    /**
     * Returns offset left.
     *
     * @returns Offset left.
     */
    get offsetLeft() {
        return this[PropertySymbol.offsetLeft];
    }
    /**
     * Returns offset top.
     *
     * @returns Offset top.
     */
    get offsetTop() {
        return this[PropertySymbol.offsetTop];
    }
    /**
     * Returns client height.
     *
     * @returns Client height.
     */
    get clientHeight() {
        return this[PropertySymbol.clientHeight];
    }
    /**
     * Returns client width.
     *
     * @returns Client width.
     */
    get clientWidth() {
        return this[PropertySymbol.clientWidth];
    }
    /**
     * Returns client left.
     *
     * @returns Client left.
     */
    get clientLeft() {
        return this[PropertySymbol.clientLeft];
    }
    /**
     * Returns client top.
     *
     * @returns Client top.
     */
    get clientTop() {
        return this[PropertySymbol.clientTop];
    }
    /**
     * Returns tab index.
     *
     * @returns Tab index.
     */
    get tabIndex() {
        const tabIndex = this.getAttribute('tabindex');
        return tabIndex !== null ? Number(tabIndex) : -1;
    }
    /**
     * Returns tab index.
     *
     * @param tabIndex Tab index.
     */
    set tabIndex(tabIndex) {
        if (tabIndex === -1) {
            this.removeAttribute('tabindex');
        }
        else {
            this.setAttribute('tabindex', String(tabIndex));
        }
    }
    /**
     * Returns inner text, which is the rendered appearance of text.
     *
     * @see https://html.spec.whatwg.org/multipage/dom.html#the-innertext-idl-attribute
     * @returns Inner text.
     */
    get innerText() {
        if (!this[PropertySymbol.isConnected]) {
            return this.textContent;
        }
        let result = '';
        for (const childNode of this[PropertySymbol.childNodes]) {
            if (childNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
                const childElement = childNode;
                const computedStyle = this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].getComputedStyle(childElement);
                if (childElement[PropertySymbol.tagName] !== 'SCRIPT' &&
                    childElement[PropertySymbol.tagName] !== 'STYLE') {
                    const display = computedStyle.display;
                    if (display !== 'none') {
                        const textTransform = computedStyle.textTransform;
                        if ((display === 'block' || display === 'flex') && result) {
                            result += '\n';
                        }
                        let text = childElement.innerText;
                        switch (textTransform) {
                            case 'uppercase':
                                text = text.toUpperCase();
                                break;
                            case 'lowercase':
                                text = text.toLowerCase();
                                break;
                            case 'capitalize':
                                text = text.replace(/(^|\s)\S/g, (l) => l.toUpperCase());
                                break;
                        }
                        result += text;
                    }
                }
            }
            else if (childNode[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
                result += childNode.textContent.replace(/[\n\r]/, '');
            }
        }
        return result;
    }
    /**
     * Sets the inner text, which is the rendered appearance of text.
     *
     * @see https://html.spec.whatwg.org/multipage/dom.html#the-innertext-idl-attribute
     * @param innerText Inner text.
     */
    set innerText(text) {
        for (const child of this[PropertySymbol.childNodes].slice()) {
            this.removeChild(child);
        }
        const texts = text.split(/[\n\r]/);
        for (let i = 0, max = texts.length; i < max; i++) {
            if (i !== 0) {
                this.appendChild(this[PropertySymbol.ownerDocument].createElement('br'));
            }
            this.appendChild(this[PropertySymbol.ownerDocument].createTextNode(texts[i]));
        }
    }
    /**
     * Returns outer text.
     *
     * @see https://html.spec.whatwg.org/multipage/dom.html#the-innertext-idl-attribute
     * @returns HTML.
     */
    get outerText() {
        return this.innerText;
    }
    /**
     * Sets outer text.
     *
     * @see https://html.spec.whatwg.org/multipage/dom.html#the-innertext-idl-attribute
     * @param text Text.
     */
    set outerText(text) {
        if (!this[PropertySymbol.parentNode]) {
            throw new DOMException("Failed to set the 'outerHTML' property on 'Element': This element has no parent node.");
        }
        const texts = text.split(/[\n\r]/);
        for (let i = 0, max = texts.length; i < max; i++) {
            if (i !== 0) {
                this[PropertySymbol.parentNode].insertBefore(this[PropertySymbol.ownerDocument].createElement('br'), this);
            }
            this[PropertySymbol.parentNode].insertBefore(this[PropertySymbol.ownerDocument].createTextNode(texts[i]), this);
        }
        this[PropertySymbol.parentNode].removeChild(this);
    }
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get style() {
        if (!this[PropertySymbol.style]) {
            this[PropertySymbol.style] = new CSSStyleDeclaration(this);
        }
        return this[PropertySymbol.style];
    }
    /**
     * Sets style.
     *
     * @param cssText Style as text.
     * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/style#setting_styles
     */
    set style(cssText) {
        this.style.cssText = typeof cssText === 'string' ? cssText : '';
    }
    /**
     * Returns data set.
     *
     * @returns Data set.
     */
    get dataset() {
        return (__classPrivateFieldSet(this, _HTMLElement_dataset, __classPrivateFieldGet(this, _HTMLElement_dataset, "f") ?? DatasetFactory.createDataset(this), "f"));
    }
    /**
     * Returns direction.
     *
     * @returns Direction.
     */
    get dir() {
        return this.getAttribute('dir') || '';
    }
    /**
     * Returns direction.
     *
     * @param direction Direction.
     */
    set dir(direction) {
        this.setAttribute('dir', direction);
    }
    /**
     * Returns hidden.
     *
     * @returns Hidden.
     */
    get hidden() {
        return this.getAttribute('hidden') !== null;
    }
    /**
     * Returns hidden.
     *
     * @param hidden Hidden.
     */
    set hidden(hidden) {
        if (!hidden) {
            this.removeAttribute('hidden');
        }
        else {
            this.setAttribute('hidden', '');
        }
    }
    /**
     * Returns inert.
     *
     * @returns Inert.
     */
    get inert() {
        return this.getAttribute('inert') !== null;
    }
    /**
     * Returns inert.
     *
     * @param inert Inert.
     */
    set inert(inert) {
        if (!inert) {
            this.removeAttribute('inert');
        }
        else {
            this.setAttribute('inert', '');
        }
    }
    /**
     * Returns language.
     *
     * @returns Language.
     */
    get lang() {
        return this.getAttribute('lang') || '';
    }
    /**
     * Returns language.
     *
     * @param language Language.
     */
    set lang(lang) {
        this.setAttribute('lang', lang);
    }
    /**
     * Returns title.
     *
     * @returns Title.
     */
    get title() {
        return this.getAttribute('title') || '';
    }
    /**
     * Returns title.
     *
     * @param title Title.
     */
    set title(title) {
        this.setAttribute('title', title);
    }
    /**
     * Triggers a click event.
     */
    click() {
        const event = new PointerEvent('click', {
            bubbles: true,
            composed: true
        });
        event[PropertySymbol.target] = this;
        event[PropertySymbol.currentTarget] = this;
        this.dispatchEvent(event);
    }
    /**
     * Triggers a blur event.
     */
    blur() {
        HTMLElementUtility.blur(this);
    }
    /**
     * Triggers a focus event.
     */
    focus() {
        HTMLElementUtility.focus(this);
    }
    /**
     * @override
     */
    [(_HTMLElement_dataset = new WeakMap(), _HTMLElement_customElementDefineCallback = new WeakMap(), _a = PropertySymbol.attributes, _b = PropertySymbol.accessKey, _c = PropertySymbol.contentEditable, _d = PropertySymbol.isContentEditable, _e = PropertySymbol.offsetHeight, _f = PropertySymbol.offsetWidth, _g = PropertySymbol.offsetLeft, _h = PropertySymbol.offsetTop, _j = PropertySymbol.clientHeight, _k = PropertySymbol.clientWidth, _l = PropertySymbol.clientLeft, _m = PropertySymbol.clientTop, _o = PropertySymbol.style, PropertySymbol.cloneNode)](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.accessKey] = this[PropertySymbol.accessKey];
        clone[PropertySymbol.contentEditable] = this[PropertySymbol.contentEditable];
        clone[PropertySymbol.isContentEditable] = this[PropertySymbol.isContentEditable];
        if (this[PropertySymbol.style]) {
            clone.style.cssText = this[PropertySymbol.style].cssText;
        }
        return clone;
    }
    /**
     * Connects this element to another element.
     *
     * @see https://html.spec.whatwg.org/multipage/dom.html#htmlelement
     * @param parentNode Parent node.
     */
    [PropertySymbol.connectToNode](parentNode = null) {
        const localName = this[PropertySymbol.localName];
        // This element can potentially be a custom element that has not been defined yet
        // Therefore we need to register a callback for when it is defined in CustomElementRegistry and replace it with the registered element (see #404)
        if (this.constructor === HTMLElement &&
            localName.includes('-') &&
            this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].customElements[PropertySymbol.callbacks]) {
            const callbacks = this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].customElements[PropertySymbol.callbacks];
            if (parentNode && !__classPrivateFieldGet(this, _HTMLElement_customElementDefineCallback, "f")) {
                const callback = () => {
                    if (this[PropertySymbol.parentNode]) {
                        const newElement = (this[PropertySymbol.ownerDocument].createElement(localName));
                        newElement[PropertySymbol.childNodes] =
                            this[PropertySymbol.childNodes];
                        newElement[PropertySymbol.children] =
                            this[PropertySymbol.children];
                        newElement[PropertySymbol.isConnected] = this[PropertySymbol.isConnected];
                        newElement[PropertySymbol.rootNode] = this[PropertySymbol.rootNode];
                        newElement[PropertySymbol.formNode] = this[PropertySymbol.formNode];
                        newElement[PropertySymbol.selectNode] = this[PropertySymbol.selectNode];
                        newElement[PropertySymbol.textAreaNode] = this[PropertySymbol.textAreaNode];
                        newElement[PropertySymbol.observers] = this[PropertySymbol.observers];
                        newElement[PropertySymbol.isValue] = this[PropertySymbol.isValue];
                        for (let i = 0, max = this[PropertySymbol.attributes].length; i < max; i++) {
                            newElement[PropertySymbol.attributes].setNamedItem(this[PropertySymbol.attributes][i]);
                        }
                        this[PropertySymbol.childNodes] = new NodeList();
                        this[PropertySymbol.children] = new HTMLCollection();
                        this[PropertySymbol.rootNode] = null;
                        this[PropertySymbol.formNode] = null;
                        this[PropertySymbol.selectNode] = null;
                        this[PropertySymbol.textAreaNode] = null;
                        this[PropertySymbol.observers] = [];
                        this[PropertySymbol.isValue] = null;
                        this[PropertySymbol.attributes] =
                            new HTMLElementNamedNodeMap(this);
                        for (let i = 0, max = this[PropertySymbol.parentNode][PropertySymbol.childNodes]
                            .length; i < max; i++) {
                            if (this[PropertySymbol.parentNode][PropertySymbol.childNodes][i] ===
                                this) {
                                this[PropertySymbol.parentNode][PropertySymbol.childNodes][i] =
                                    newElement;
                                break;
                            }
                        }
                        if (this[PropertySymbol.parentNode][PropertySymbol.children]) {
                            for (let i = 0, max = this[PropertySymbol.parentNode][PropertySymbol.children]
                                .length; i < max; i++) {
                                if (this[PropertySymbol.parentNode][PropertySymbol.children][i] ===
                                    this) {
                                    this[PropertySymbol.parentNode][PropertySymbol.children][i] =
                                        newElement;
                                    break;
                                }
                            }
                        }
                        if (newElement[PropertySymbol.isConnected] && newElement.connectedCallback) {
                            const result = newElement.connectedCallback();
                            /**
                             * It is common to import dependencies in the connectedCallback() method of web components.
                             * As Happy DOM doesn't have support for dynamic imports yet, this is a temporary solution to wait for imports in connectedCallback().
                             *
                             * @see https://github.com/capricorn86/happy-dom/issues/1442
                             */
                            if (result instanceof Promise) {
                                const asyncTaskManager = this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow][PropertySymbol.asyncTaskManager];
                                const taskID = asyncTaskManager.startTask();
                                result
                                    .then(() => asyncTaskManager.endTask(taskID))
                                    .catch(() => asyncTaskManager.endTask(taskID));
                            }
                        }
                        this[PropertySymbol.connectToNode](null);
                    }
                };
                callbacks[localName] = callbacks[localName] || [];
                callbacks[localName].push(callback);
                __classPrivateFieldSet(this, _HTMLElement_customElementDefineCallback, callback, "f");
            }
            else if (!parentNode && callbacks[localName] && __classPrivateFieldGet(this, _HTMLElement_customElementDefineCallback, "f")) {
                const index = callbacks[localName].indexOf(__classPrivateFieldGet(this, _HTMLElement_customElementDefineCallback, "f"));
                if (index !== -1) {
                    callbacks[localName].splice(index, 1);
                }
                if (!callbacks[localName].length) {
                    delete callbacks[localName];
                }
                __classPrivateFieldSet(this, _HTMLElement_customElementDefineCallback, null, "f");
            }
        }
        super[PropertySymbol.connectToNode](parentNode);
    }
}
//# sourceMappingURL=HTMLElement.js.map