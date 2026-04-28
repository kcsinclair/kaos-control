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
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _HTMLSelectElement_instances, _HTMLSelectElement_getDisplaySize, _a, _b, _c, _d, _e, _f;
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const ValidityState_js_1 = __importDefault(require("../../validity-state/ValidityState.cjs"));
const HTMLOptionsCollection_js_1 = __importDefault(require("./HTMLOptionsCollection.cjs"));
const Event_js_1 = __importDefault(require("../../event/Event.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../node/NodeTypeEnum.cjs"));
const HTMLLabelElementUtility_js_1 = __importDefault(require("../html-label-element/HTMLLabelElementUtility.cjs"));
const HTMLSelectElementNamedNodeMap_js_1 = __importDefault(require("./HTMLSelectElementNamedNodeMap.cjs"));
const HTMLCollection_js_1 = __importDefault(require("../element/HTMLCollection.cjs"));
/**
 * HTML Select Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLSelectElement.
 */
class HTMLSelectElement extends HTMLElement_js_1.default {
    constructor() {
        super(...arguments);
        _HTMLSelectElement_instances.add(this);
        // Internal properties.
        this[_a] = new HTMLSelectElementNamedNodeMap_js_1.default(this);
        this[_b] = '';
        this[_c] = new ValidityState_js_1.default(this);
        this[_d] = this;
        this[_e] = 0;
        this[_f] = new HTMLOptionsCollection_js_1.default(this);
        // Events
        this.onchange = null;
        this.oninput = null;
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
     * Returns options.
     *
     * @returns Options.
     */
    get options() {
        return this[PropertySymbol.options];
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
     * Returns multiple.
     *
     * @returns Multiple.
     */
    get multiple() {
        return this.getAttribute('multiple') !== null;
    }
    /**
     * Sets multiple.
     *
     * @param multiple Multiple.
     */
    set multiple(multiple) {
        if (!multiple) {
            this.removeAttribute('multiple');
        }
        else {
            this.setAttribute('multiple', '');
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
     * Returns type.
     *
     * @returns type.
     */
    get type() {
        return this.hasAttributeNS(null, 'multiple') ? 'select-multiple' : 'select-one';
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        for (let i = 0, max = this[PropertySymbol.options].length; i < max; i++) {
            const option = this[PropertySymbol.options][i];
            if (option[PropertySymbol.selectedness]) {
                return option.value;
            }
        }
        return '';
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        for (let i = 0, max = this[PropertySymbol.options].length; i < max; i++) {
            const option = this[PropertySymbol.options][i];
            if (option.value === value) {
                option[PropertySymbol.selectedness] = true;
                option[PropertySymbol.dirtyness] = true;
            }
            else {
                option[PropertySymbol.selectedness] = false;
            }
        }
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get selectedIndex() {
        for (let i = 0, max = this[PropertySymbol.options].length; i < max; i++) {
            if (this[PropertySymbol.options][i][PropertySymbol.selectedness]) {
                return i;
            }
        }
        return -1;
    }
    /**
     * Sets value.
     *
     * @param selectedIndex Selected index.
     */
    set selectedIndex(selectedIndex) {
        if (typeof selectedIndex === 'number' && !isNaN(selectedIndex)) {
            for (let i = 0, max = this[PropertySymbol.options].length; i < max; i++) {
                this[PropertySymbol.options][i][PropertySymbol.selectedness] = false;
            }
            const selectedOption = this[PropertySymbol.options][selectedIndex];
            if (selectedOption) {
                selectedOption[PropertySymbol.selectedness] = true;
                selectedOption[PropertySymbol.dirtyness] = true;
            }
        }
    }
    /**
     * Returns selected options.
     *
     * @returns HTMLCollection.
     */
    get selectedOptions() {
        const selectedOptions = new HTMLCollection_js_1.default();
        for (let i = 0, max = this[PropertySymbol.options].length; i < max; i++) {
            if (this[PropertySymbol.options][i][PropertySymbol.selectedness]) {
                selectedOptions.push(this[PropertySymbol.options][i]);
            }
        }
        return selectedOptions;
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
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form() {
        return this[PropertySymbol.formNode];
    }
    /**
     * Returns "true" if it will validate.
     *
     * @returns "true" if it will validate.
     */
    get willValidate() {
        return (this.type !== 'hidden' &&
            this.type !== 'reset' &&
            this.type !== 'button' &&
            !this.disabled &&
            !this['readOnly']);
    }
    /**
     * Returns item from options collection by index.
     *
     * @param index Index.
     */
    item(index) {
        return this[PropertySymbol.options].item(index);
    }
    /**
     * Adds new option to options collection.
     *
     * @param element HTMLOptionElement to add.
     * @param before HTMLOptionElement or index number.
     */
    add(element, before) {
        this[PropertySymbol.options].add(element, before);
    }
    /**
     * Removes indexed element from collection or the select element.
     *
     * @param [index] Index.
     */
    remove(index) {
        if (typeof index === 'number') {
            this[PropertySymbol.options].remove(index);
        }
        else {
            super.remove();
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
        const valid = this.disabled || this[PropertySymbol.validity].valid;
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
     * Updates option item.
     *
     * Based on:
     * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/nodes/HTMLSelectElement-impl.js
     *
     * @see https://html.spec.whatwg.org/multipage/form-elements.html#selectedness-setting-algorithm
     * @param [selectedOption] Selected option.
     */
    [(_HTMLSelectElement_instances = new WeakSet(), _a = PropertySymbol.attributes, _b = PropertySymbol.validationMessage, _c = PropertySymbol.validity, _d = PropertySymbol.selectNode, _e = PropertySymbol.length, _f = PropertySymbol.options, PropertySymbol.updateOptionItems)](selectedOption) {
        const optionElements = this.getElementsByTagName('option');
        if (optionElements.length < this[PropertySymbol.options].length) {
            this[PropertySymbol.options].splice(this[PropertySymbol.options].length - 1, this[PropertySymbol.options].length - optionElements.length);
            for (let i = optionElements.length - 1, max = this[PropertySymbol.length]; i < max; i++) {
                delete this[i];
            }
        }
        const isMultiple = this.hasAttributeNS(null, 'multiple');
        const selected = [];
        for (let i = 0; i < optionElements.length; i++) {
            this[PropertySymbol.options][i] = optionElements[i];
            this[i] = optionElements[i];
            if (!isMultiple) {
                if (selectedOption) {
                    optionElements[i][PropertySymbol.selectedness] =
                        optionElements[i] === selectedOption;
                }
                if (optionElements[i][PropertySymbol.selectedness]) {
                    selected.push(optionElements[i]);
                }
            }
        }
        this[PropertySymbol.length] = optionElements.length;
        const size = __classPrivateFieldGet(this, _HTMLSelectElement_instances, "m", _HTMLSelectElement_getDisplaySize).call(this);
        if (size === 1 && !selected.length) {
            for (let i = 0, max = optionElements.length; i < max; i++) {
                const option = optionElements[i];
                let disabled = option.hasAttributeNS(null, 'disabled');
                const parentNode = option[PropertySymbol.parentNode];
                if (parentNode &&
                    parentNode[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.elementNode &&
                    parentNode[PropertySymbol.tagName] === 'OPTGROUP' &&
                    parentNode.hasAttributeNS(null, 'disabled')) {
                    disabled = true;
                }
                if (!disabled) {
                    option[PropertySymbol.selectedness] = true;
                    break;
                }
            }
        }
        else if (selected.length >= 2) {
            for (let i = 0, max = optionElements.length; i < max; i++) {
                optionElements[i][PropertySymbol.selectedness] =
                    i === selected.length - 1;
            }
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
_HTMLSelectElement_getDisplaySize = function _HTMLSelectElement_getDisplaySize() {
    if (this.hasAttributeNS(null, 'size')) {
        const size = parseInt(this.getAttribute('size'));
        if (!isNaN(size) && size >= 0) {
            return size;
        }
    }
    return this.hasAttributeNS(null, 'multiple') ? 4 : 1;
};
exports.default = HTMLSelectElement;
//# sourceMappingURL=HTMLSelectElement.cjs.map