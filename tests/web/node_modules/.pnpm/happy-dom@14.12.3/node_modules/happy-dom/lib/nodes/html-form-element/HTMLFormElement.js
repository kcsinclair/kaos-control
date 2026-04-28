var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _HTMLFormElement_instances, _HTMLFormElement_browserFrame, _HTMLFormElement_submit, _a, _b, _c;
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Event from '../../event/Event.js';
import SubmitEvent from '../../event/events/SubmitEvent.js';
import HTMLFormControlsCollection from './HTMLFormControlsCollection.js';
import Node from '../node/Node.js';
import BrowserFrameNavigator from '../../browser/utilities/BrowserFrameNavigator.js';
import FormData from '../../form-data/FormData.js';
import Element from '../element/Element.js';
/**
 * HTML Form Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLFormElement.
 */
class HTMLFormElement extends HTMLElement {
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     */
    constructor(browserFrame) {
        super();
        _HTMLFormElement_instances.add(this);
        // Internal properties.
        this[_a] = new HTMLFormControlsCollection();
        this[_b] = 0;
        this[_c] = this;
        // Events
        this.onformdata = null;
        this.onreset = null;
        this.onsubmit = null;
        // Private properties
        _HTMLFormElement_browserFrame.set(this, void 0);
        __classPrivateFieldSet(this, _HTMLFormElement_browserFrame, browserFrame, "f");
    }
    /**
     * Returns elements.
     *
     * @returns Elements.
     */
    get elements() {
        return this[PropertySymbol.elements];
    }
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length() {
        return this[PropertySymbol.length];
    }
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this.getAttribute('name') || '';
    }
    /**
     * Sets name.
     *
     * @param name Name.
     */
    set name(name) {
        this.setAttribute('name', name);
    }
    /**
     * Returns method.
     *
     * @returns Method.
     */
    get method() {
        return this.getAttribute('method') || 'get';
    }
    /**
     * Sets method.
     *
     * @param method Method.
     */
    set method(method) {
        this.setAttribute('method', method);
    }
    /**
     * Returns target.
     *
     * @returns Target.
     */
    get target() {
        return this.getAttribute('target') || '';
    }
    /**
     * Sets target.
     *
     * @param target Target.
     */
    set target(target) {
        this.setAttribute('target', target);
    }
    /**
     * Returns action.
     *
     * @returns Action.
     */
    get action() {
        if (!this.hasAttribute('action')) {
            return this[PropertySymbol.ownerDocument].location.href;
        }
        try {
            return new URL(this.getAttribute('action'), this[PropertySymbol.ownerDocument].location.href)
                .href;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets action.
     *
     * @param action Action.
     */
    set action(action) {
        this.setAttribute('action', action);
    }
    /**
     * Returns encoding.
     *
     * @returns Encoding.
     */
    get encoding() {
        return this.getAttribute('encoding') || '';
    }
    /**
     * Sets encoding.
     *
     * @param encoding Encoding.
     */
    set encoding(encoding) {
        this.setAttribute('encoding', encoding);
    }
    /**
     * Returns enctype.
     *
     * @returns Enctype.
     */
    get enctype() {
        return this.getAttribute('enctype') || '';
    }
    /**
     * Sets enctype.
     *
     * @param enctype Enctype.
     */
    set enctype(enctype) {
        this.setAttribute('enctype', enctype);
    }
    /**
     * Returns autocomplete.
     *
     * @returns Autocomplete.
     */
    get autocomplete() {
        return this.getAttribute('autocomplete') || '';
    }
    /**
     * Sets autocomplete.
     *
     * @param autocomplete Autocomplete.
     */
    set autocomplete(autocomplete) {
        this.setAttribute('autocomplete', autocomplete);
    }
    /**
     * Returns accept charset.
     *
     * @returns Accept charset.
     */
    get acceptCharset() {
        return this.getAttribute('acceptcharset') || '';
    }
    /**
     * Sets accept charset.
     *
     * @param acceptCharset Accept charset.
     */
    set acceptCharset(acceptCharset) {
        this.setAttribute('acceptcharset', acceptCharset);
    }
    /**
     * Returns no validate.
     *
     * @returns No validate.
     */
    get noValidate() {
        return this.getAttribute('novalidate') !== null;
    }
    /**
     * Sets no validate.
     *
     * @param noValidate No validate.
     */
    set noValidate(noValidate) {
        if (!noValidate) {
            this.removeAttribute('novalidate');
        }
        else {
            this.setAttribute('novalidate', '');
        }
    }
    /**
     * Submits form. No submit event is raised. In particular, the form's "submit" event handler is not run.
     */
    submit() {
        __classPrivateFieldGet(this, _HTMLFormElement_instances, "m", _HTMLFormElement_submit).call(this);
    }
    /**
     * Submits form, reports validity and raises submit event.
     *
     * @param [submitter] Submitter.
     */
    requestSubmit(submitter) {
        const noValidate = submitter?.formNoValidate || this.noValidate;
        if (noValidate || this.checkValidity()) {
            this.dispatchEvent(new SubmitEvent('submit', { bubbles: true, cancelable: true, submitter: submitter || this }));
            __classPrivateFieldGet(this, _HTMLFormElement_instances, "m", _HTMLFormElement_submit).call(this, submitter);
        }
    }
    /**
     * Resets form.
     */
    reset() {
        for (const element of this[PropertySymbol.elements]) {
            if (element[PropertySymbol.tagName] === 'INPUT' ||
                element[PropertySymbol.tagName] === 'TEXTAREA') {
                element[PropertySymbol.value] = null;
                element[PropertySymbol.checked] = null;
            }
            else if (element[PropertySymbol.tagName] === 'TEXTAREA') {
                element[PropertySymbol.value] = null;
            }
            else if (element[PropertySymbol.tagName] === 'SELECT') {
                let hasSelectedAttribute = false;
                for (const option of element.options) {
                    if (option.hasAttribute('selected')) {
                        hasSelectedAttribute = true;
                        option.selected = true;
                        break;
                    }
                }
                if (!hasSelectedAttribute && element.options.length > 0) {
                    element.options[0].selected = true;
                }
            }
        }
        this.dispatchEvent(new Event('reset', { bubbles: true, cancelable: true }));
    }
    /**
     * Checks validity.
     *
     * @returns "true" if validation does'nt fail.
     */
    checkValidity() {
        const radioValidationState = {};
        let isFormValid = true;
        for (const element of this[PropertySymbol.elements]) {
            if (element[PropertySymbol.tagName] === 'INPUT' && element.type === 'radio' && element.name) {
                if (!radioValidationState[element.name]) {
                    radioValidationState[element.name] = true;
                    if (!element.checkValidity()) {
                        isFormValid = false;
                    }
                }
            }
            else if (!element.checkValidity()) {
                isFormValid = false;
            }
        }
        return isFormValid;
    }
    /**
     * Reports validity.
     *
     * @returns "true" if validation does'nt fail.
     */
    reportValidity() {
        return this.checkValidity();
    }
    /**
     * @override
     */
    [(_HTMLFormElement_browserFrame = new WeakMap(), _HTMLFormElement_instances = new WeakSet(), _a = PropertySymbol.elements, _b = PropertySymbol.length, _c = PropertySymbol.formNode, PropertySymbol.cloneNode)](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
    /**
     * Appends a form control item.
     *
     * @param node Node.
     * @param name Name
     */
    [PropertySymbol.appendFormControlItem](node, name) {
        const elements = this[PropertySymbol.elements];
        if (!elements.includes(node)) {
            this[elements.length] = node;
            elements.push(node);
            this[PropertySymbol.length] = elements.length;
        }
        elements[PropertySymbol.appendNamedItem](node, name);
        if (this[PropertySymbol.isValidPropertyName](name)) {
            this[name] = elements[name];
        }
    }
    /**
     * Remove a form control item.
     *
     * @param node Node.
     * @param name Name.
     */
    [PropertySymbol.removeFormControlItem](node, name) {
        const elements = this[PropertySymbol.elements];
        const index = elements.indexOf(node);
        if (index !== -1) {
            elements.splice(index, 1);
            for (let i = index; i < this[PropertySymbol.length]; i++) {
                this[i] = this[i + 1];
            }
            delete this[this[PropertySymbol.length] - 1];
            this[PropertySymbol.length]--;
        }
        elements[PropertySymbol.removeNamedItem](node, name);
        if (this[PropertySymbol.isValidPropertyName](name)) {
            if (elements[name]) {
                this[name] = elements[name];
            }
            else {
                delete this[name];
            }
        }
    }
    /**
     * Returns "true" if the property name is valid.
     *
     * @param name Name.
     * @returns True if the property name is valid.
     */
    [PropertySymbol.isValidPropertyName](name) {
        return (!!name &&
            !HTMLFormElement.prototype.hasOwnProperty(name) &&
            !HTMLElement.prototype.hasOwnProperty(name) &&
            !Element.prototype.hasOwnProperty(name) &&
            !Node.prototype.hasOwnProperty(name) &&
            (isNaN(Number(name)) || name.includes('.')));
    }
}
_HTMLFormElement_submit = function _HTMLFormElement_submit(submitter) {
    const action = submitter?.hasAttribute('formaction')
        ? submitter?.formAction || this.action
        : this.action;
    if (!action) {
        // The URL is invalid when the action is empty.
        // This is what Chrome does when the URL is invalid.
        this[PropertySymbol.ownerDocument].location.hash = '#blocked';
        return;
    }
    const method = submitter?.formMethod || this.method;
    const formData = new FormData(this);
    let targetFrame;
    switch (submitter?.formTarget || this.target) {
        default:
        case '_self':
            targetFrame = __classPrivateFieldGet(this, _HTMLFormElement_browserFrame, "f");
            break;
        case '_top':
            targetFrame = __classPrivateFieldGet(this, _HTMLFormElement_browserFrame, "f").page.mainFrame;
            break;
        case '_parent':
            targetFrame = __classPrivateFieldGet(this, _HTMLFormElement_browserFrame, "f").parentFrame ?? __classPrivateFieldGet(this, _HTMLFormElement_browserFrame, "f");
            break;
        case '_blank':
            const newPage = __classPrivateFieldGet(this, _HTMLFormElement_browserFrame, "f").page.context.newPage();
            targetFrame = newPage.mainFrame;
            targetFrame[PropertySymbol.openerFrame] = __classPrivateFieldGet(this, _HTMLFormElement_browserFrame, "f");
            break;
    }
    if (method === 'get') {
        const url = new URL(action);
        for (const [key, value] of formData) {
            if (typeof value === 'string') {
                url.searchParams.append(key, value);
            }
        }
        BrowserFrameNavigator.navigate({
            windowClass: (this[PropertySymbol.ownerDocument][PropertySymbol.defaultView].constructor),
            frame: targetFrame,
            url: url.href,
            goToOptions: {
                referrer: __classPrivateFieldGet(this, _HTMLFormElement_browserFrame, "f").page.mainFrame.window.location.origin
            }
        });
        return;
    }
    BrowserFrameNavigator.navigate({
        windowClass: (this[PropertySymbol.ownerDocument][PropertySymbol.defaultView].constructor),
        frame: targetFrame,
        method: method,
        url: action,
        formData,
        goToOptions: {
            referrer: __classPrivateFieldGet(this, _HTMLFormElement_browserFrame, "f").page.mainFrame.window.location.origin
        }
    });
};
export default HTMLFormElement;
//# sourceMappingURL=HTMLFormElement.js.map