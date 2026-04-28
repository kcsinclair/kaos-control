"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _HTMLTextAreaElement_selectionStart, _HTMLTextAreaElement_selectionEnd, _HTMLTextAreaElement_selectionDirection, _a, _b, _c, _d, _e;
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../../event/Event.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const DOMException_js_1 = __importDefault(require("../../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../../exception/DOMExceptionNameEnum.cjs"));
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const HTMLInputElementSelectionDirectionEnum_js_1 = __importDefault(require("../html-input-element/HTMLInputElementSelectionDirectionEnum.cjs"));
const HTMLInputElementSelectionModeEnum_js_1 = __importDefault(require("../html-input-element/HTMLInputElementSelectionModeEnum.cjs"));
const ValidityState_js_1 = __importDefault(require("../../validity-state/ValidityState.cjs"));
const HTMLLabelElementUtility_js_1 = __importDefault(require("../html-label-element/HTMLLabelElementUtility.cjs"));
const HTMLTextAreaElementNamedNodeMap_js_1 = __importDefault(require("./HTMLTextAreaElementNamedNodeMap.cjs"));
/**
 * HTML Text Area Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLTextAreaElement.
 */
class HTMLTextAreaElement extends HTMLElement_js_1.default {
    constructor() {
        super(...arguments);
        this.type = 'textarea';
        // Events
        this.oninput = null;
        this.onselectionchange = null;
        // Internal properties
        this[_a] = new HTMLTextAreaElementNamedNodeMap_js_1.default(this);
        this[_b] = '';
        this[_c] = new ValidityState_js_1.default(this);
        this[_d] = null;
        this[_e] = this;
        // Private properties
        _HTMLTextAreaElement_selectionStart.set(this, null);
        _HTMLTextAreaElement_selectionEnd.set(this, null);
        _HTMLTextAreaElement_selectionDirection.set(this, HTMLInputElementSelectionDirectionEnum_js_1.default.none);
    }
    /**
     * Returns validation message.
     *
     * @returns Validation message.
     */
    get validationMessage() {
        return this[PropertySymbol.validationMessage];
    }
    /**
     * Returns validity.
     *
     * @returns Validity.
     */
    get validity() {
        return this[PropertySymbol.validity];
    }
    /**
     * Returns the default value.
     *
     * @returns Default value.
     */
    get defaultValue() {
        return this.textContent;
    }
    /**
     * Sets the default value.
     *
     * @param defaultValue Default value.
     */
    set defaultValue(defaultValue) {
        this.textContent = defaultValue;
    }
    /**
     * Returns minlength.
     *
     * @returns Min length.
     */
    get minLength() {
        const minLength = this.getAttribute('minlength');
        if (minLength !== null) {
            return parseInt(minLength);
        }
        return -1;
    }
    /**
     * Sets minlength.
     *
     * @param minLength Min length.
     */
    set minLength(minlength) {
        this.setAttribute('minlength', String(minlength));
    }
    /**
     * Returns maxlength.
     *
     * @returns Max length.
     */
    get maxLength() {
        const maxLength = this.getAttribute('maxlength');
        if (maxLength !== null) {
            return parseInt(maxLength);
        }
        return -1;
    }
    /**
     * Sets maxlength.
     *
     * @param maxlength Max length.
     */
    set maxLength(maxLength) {
        this.setAttribute('maxlength', String(maxLength));
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
     * Returns placeholder.
     *
     * @returns Placeholder.
     */
    get placeholder() {
        return this.getAttribute('placeholder') || '';
    }
    /**
     * Sets placeholder.
     *
     * @param placeholder Placeholder.
     */
    set placeholder(placeholder) {
        this.setAttribute('placeholder', placeholder);
    }
    /**
     * Returns inputmode.
     *
     * @returns Inputmode.
     */
    get inputmode() {
        return this.getAttribute('inputmode') || '';
    }
    /**
     * Sets inputmode.
     *
     * @param inputmode Inputmode.
     */
    set inputmode(inputmode) {
        this.setAttribute('inputmode', inputmode);
    }
    /**
     * Returns cols.
     *
     * @returns Cols.
     */
    get cols() {
        return this.getAttribute('cols') || '';
    }
    /**
     * Sets cols.
     *
     * @param cols Cols.
     */
    set cols(cols) {
        this.setAttribute('cols', cols);
    }
    /**
     * Returns rows.
     *
     * @returns Rows.
     */
    get rows() {
        return this.getAttribute('rows') || '';
    }
    /**
     * Sets rows.
     *
     * @param rows Rows.
     */
    set rows(rows) {
        this.setAttribute('rows', rows);
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
     * Returns readOnly.
     *
     * @returns ReadOnly.
     */
    get readOnly() {
        return this.getAttribute('readonly') !== null;
    }
    /**
     * Sets readOnly.
     *
     * @param readOnly ReadOnly.
     */
    set readOnly(readOnly) {
        if (!readOnly) {
            this.removeAttribute('readonly');
        }
        else {
            this.setAttribute('readonly', '');
        }
    }
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled() {
        return this.getAttribute('disabled') !== null;
    }
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled) {
        if (!disabled) {
            this.removeAttribute('disabled');
        }
        else {
            this.setAttribute('disabled', '');
        }
    }
    /**
     * Returns autofocus.
     *
     * @returns Autofocus.
     */
    get autofocus() {
        return this.getAttribute('autofocus') !== null;
    }
    /**
     * Sets autofocus.
     *
     * @param autofocus Autofocus.
     */
    set autofocus(autofocus) {
        if (!autofocus) {
            this.removeAttribute('autofocus');
        }
        else {
            this.setAttribute('autofocus', '');
        }
    }
    /**
     * Returns required.
     *
     * @returns Required.
     */
    get required() {
        return this.getAttribute('required') !== null;
    }
    /**
     * Sets required.
     *
     * @param required Required.
     */
    set required(required) {
        if (!required) {
            this.removeAttribute('required');
        }
        else {
            this.setAttribute('required', '');
        }
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        if (this[PropertySymbol.value] === null) {
            return this.textContent;
        }
        return this[PropertySymbol.value];
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        const oldValue = this[PropertySymbol.value];
        this[PropertySymbol.value] = value;
        if (oldValue !== this[PropertySymbol.value]) {
            __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionStart, this[PropertySymbol.value].length, "f");
            __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionEnd, this[PropertySymbol.value].length, "f");
            __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionDirection, HTMLInputElementSelectionDirectionEnum_js_1.default.none, "f");
        }
    }
    /**
     * Returns selection start.
     *
     * @returns Selection start.
     */
    get selectionStart() {
        if (__classPrivateFieldGet(this, _HTMLTextAreaElement_selectionStart, "f") === null) {
            return this.value.length;
        }
        return __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionStart, "f");
    }
    /**
     * Sets selection start.
     *
     * @param start Start.
     */
    set selectionStart(start) {
        this.setSelectionRange(start, Math.max(start, this.selectionEnd), __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionDirection, "f"));
    }
    /**
     * Returns selection end.
     *
     * @returns Selection end.
     */
    get selectionEnd() {
        if (__classPrivateFieldGet(this, _HTMLTextAreaElement_selectionEnd, "f") === null) {
            return this.value.length;
        }
        return __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionEnd, "f");
    }
    /**
     * Sets selection end.
     *
     * @param end End.
     */
    set selectionEnd(end) {
        this.setSelectionRange(this.selectionStart, end, __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionDirection, "f"));
    }
    /**
     * Returns selection direction.
     *
     * @returns Selection direction.
     */
    get selectionDirection() {
        return __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionDirection, "f");
    }
    /**
     * Sets selection direction.
     *
     * @param direction Direction.
     */
    set selectionDirection(direction) {
        this.setSelectionRange(this.selectionStart, this.selectionEnd, direction);
    }
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form() {
        return this[PropertySymbol.formNode];
    }
    /**
     * Returns text length.
     *
     * @param Text Length.
     */
    get textLength() {
        return this.value.length;
    }
    /**
     * Returns the associated label elements.
     *
     * @returns Label elements.
     */
    get labels() {
        return HTMLLabelElementUtility_js_1.default.getAssociatedLabelElements(this);
    }
    /**
     * Selects the text.
     */
    select() {
        __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionStart, 0, "f");
        __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionEnd, this.value.length, "f");
        __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionDirection, HTMLInputElementSelectionDirectionEnum_js_1.default.none, "f");
        this.dispatchEvent(new Event_js_1.default('select', { bubbles: true, cancelable: true }));
    }
    /**
     * Set selection range.
     *
     * @param start Start.
     * @param end End.
     * @param [direction="none"] Direction.
     */
    setSelectionRange(start, end, direction = 'none') {
        __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionEnd, Math.min(end, this.value.length), "f");
        __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionStart, Math.min(start, this.selectionEnd), "f");
        __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionDirection, direction === HTMLInputElementSelectionDirectionEnum_js_1.default.forward ||
            direction === HTMLInputElementSelectionDirectionEnum_js_1.default.backward
            ? direction
            : HTMLInputElementSelectionDirectionEnum_js_1.default.none, "f");
        this.dispatchEvent(new Event_js_1.default('select', { bubbles: true, cancelable: true }));
    }
    /**
     * Set range text.
     *
     * @param replacement Replacement.
     * @param [start] Start.
     * @param [end] End.
     * @param [direction] Direction.
     * @param selectionMode
     */
    setRangeText(replacement, start = null, end = null, selectionMode = HTMLInputElementSelectionModeEnum_js_1.default.preserve) {
        if (start === null) {
            start = __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionStart, "f");
        }
        if (end === null) {
            end = __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionEnd, "f");
        }
        if (start > end) {
            throw new DOMException_js_1.default('The index is not in the allowed range.', DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        start = Math.min(start, this.value.length);
        end = Math.min(end, this.value.length);
        const val = this.value;
        let selectionStart = __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionStart, "f");
        let selectionEnd = __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionEnd, "f");
        this.value = val.slice(0, start) + replacement + val.slice(end);
        const newEnd = start + this.value.length;
        switch (selectionMode) {
            case HTMLInputElementSelectionModeEnum_js_1.default.select:
                this.setSelectionRange(start, newEnd);
                break;
            case HTMLInputElementSelectionModeEnum_js_1.default.start:
                this.setSelectionRange(start, start);
                break;
            case HTMLInputElementSelectionModeEnum_js_1.default.end:
                this.setSelectionRange(newEnd, newEnd);
                break;
            default:
                const delta = replacement.length - (end - start);
                if (selectionStart > end) {
                    selectionStart += delta;
                }
                else if (selectionStart > start) {
                    selectionStart = start;
                }
                if (selectionEnd > end) {
                    selectionEnd += delta;
                }
                else if (selectionEnd > start) {
                    selectionEnd = newEnd;
                }
                this.setSelectionRange(selectionStart, selectionEnd);
                break;
        }
    }
    /**
     * Sets validation message.
     *
     * @param message Message.
     */
    setCustomValidity(message) {
        this[PropertySymbol.validationMessage] = String(message);
    }
    /**
     * Checks validity.
     *
     * @returns "true" if the field is valid.
     */
    checkValidity() {
        const valid = this.disabled || this.readOnly || this[PropertySymbol.validity].valid;
        if (!valid) {
            this.dispatchEvent(new Event_js_1.default('invalid', { bubbles: true, cancelable: true }));
        }
        return valid;
    }
    /**
     * Reports validity.
     *
     * @returns "true" if the field is valid.
     */
    reportValidity() {
        return this.checkValidity();
    }
    /**
     * @override
     */
    [(_HTMLTextAreaElement_selectionStart = new WeakMap(), _HTMLTextAreaElement_selectionEnd = new WeakMap(), _HTMLTextAreaElement_selectionDirection = new WeakMap(), _a = PropertySymbol.attributes, _b = PropertySymbol.validationMessage, _c = PropertySymbol.validity, _d = PropertySymbol.value, _e = PropertySymbol.textAreaNode, PropertySymbol.cloneNode)](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.value] = this[PropertySymbol.value];
        __classPrivateFieldSet(clone, _HTMLTextAreaElement_selectionStart, __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionStart, "f"), "f");
        __classPrivateFieldSet(clone, _HTMLTextAreaElement_selectionEnd, __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionEnd, "f"), "f");
        __classPrivateFieldSet(clone, _HTMLTextAreaElement_selectionDirection, __classPrivateFieldGet(this, _HTMLTextAreaElement_selectionDirection, "f"), "f");
        return clone;
    }
    /**
     * Resets selection.
     */
    [PropertySymbol.resetSelection]() {
        if (this[PropertySymbol.value] === null) {
            __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionStart, null, "f");
            __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionEnd, null, "f");
            __classPrivateFieldSet(this, _HTMLTextAreaElement_selectionDirection, HTMLInputElementSelectionDirectionEnum_js_1.default.none, "f");
        }
    }
    /**
     * @override
     */
    [PropertySymbol.connectToNode](parentNode = null) {
        const oldFormNode = this[PropertySymbol.formNode];
        super[PropertySymbol.connectToNode](parentNode);
        if (oldFormNode !== this[PropertySymbol.formNode]) {
            if (oldFormNode) {
                oldFormNode[PropertySymbol.removeFormControlItem](this, this.name);
                oldFormNode[PropertySymbol.removeFormControlItem](this, this.id);
            }
            if (this[PropertySymbol.formNode]) {
                this[PropertySymbol.formNode][PropertySymbol.appendFormControlItem](this, this.name);
                this[PropertySymbol.formNode][PropertySymbol.appendFormControlItem](this, this.id);
            }
        }
    }
}
exports.default = HTMLTextAreaElement;
//# sourceMappingURL=HTMLTextAreaElement.cjs.map