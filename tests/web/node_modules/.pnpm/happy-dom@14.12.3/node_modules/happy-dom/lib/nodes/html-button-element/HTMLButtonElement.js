var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _HTMLButtonElement_instances, _HTMLButtonElement_sanitizeType, _a, _b, _c;
import Event from '../../event/Event.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import ValidityState from '../../validity-state/ValidityState.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLLabelElementUtility from '../html-label-element/HTMLLabelElementUtility.js';
import HTMLButtonElementNamedNodeMap from './HTMLButtonElementNamedNodeMap.js';
import { URL } from 'url';
import MouseEvent from '../../event/events/MouseEvent.js';
const BUTTON_TYPES = ['submit', 'reset', 'button', 'menu'];
/**
 * HTML Button Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLButtonElement.
 */
class HTMLButtonElement extends HTMLElement {
    constructor() {
        super(...arguments);
        _HTMLButtonElement_instances.add(this);
        this[_a] = new HTMLButtonElementNamedNodeMap(this);
        this[_b] = '';
        this[_c] = new ValidityState(this);
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
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        return this.getAttribute('value');
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        this.setAttribute('value', value);
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
     * Returns type
     *
     * @returns Type
     */
    get type() {
        return __classPrivateFieldGet(this, _HTMLButtonElement_instances, "m", _HTMLButtonElement_sanitizeType).call(this, this.getAttribute('type'));
    }
    /**
     * Sets type
     *
     * @param v Type
     */
    set type(v) {
        this.setAttribute('type', __classPrivateFieldGet(this, _HTMLButtonElement_instances, "m", _HTMLButtonElement_sanitizeType).call(this, v));
    }
    /**
     * Returns form action.
     *
     * @returns Form action.
     */
    get formAction() {
        if (!this.hasAttribute('formaction')) {
            return this[PropertySymbol.ownerDocument].location.href;
        }
        try {
            return new URL(this.getAttribute('formaction'), this[PropertySymbol.ownerDocument].location.href).href;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets form action.
     *
     * @param formAction Form action.
     */
    set formAction(formAction) {
        this.setAttribute('formaction', formAction);
    }
    /**
     * Returns form enctype.
     *
     * @returns Form enctype.
     */
    get formEnctype() {
        return this.getAttribute('formenctype') || '';
    }
    /**
     * Sets form enctype.
     *
     * @param formEnctype Form enctype.
     */
    set formEnctype(formEnctype) {
        this.setAttribute('formenctype', formEnctype);
    }
    /**
     * Returns form method.
     *
     * @returns Form method.
     */
    get formMethod() {
        return this.getAttribute('formmethod') || '';
    }
    /**
     * Sets form method.
     *
     * @param formMethod Form method.
     */
    set formMethod(formMethod) {
        this.setAttribute('formmethod', formMethod);
    }
    /**
     * Returns no validate.
     *
     * @returns No validate.
     */
    get formNoValidate() {
        return this.getAttribute('formnovalidate') !== null;
    }
    /**
     * Sets no validate.
     *
     * @param formNoValidate No validate.
     */
    set formNoValidate(formNoValidate) {
        if (!formNoValidate) {
            this.removeAttribute('formnovalidate');
        }
        else {
            this.setAttribute('formnovalidate', '');
        }
    }
    /**
     * Returns form target.
     *
     * @returns Form target.
     */
    get formTarget() {
        return this.getAttribute('formtarget') || '';
    }
    /**
     * Sets form target.
     *
     * @param formTarget Form target.
     */
    set formTarget(formTarget) {
        this.setAttribute('formtarget', formTarget);
    }
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form() {
        if (this[PropertySymbol.formNode]) {
            return this[PropertySymbol.formNode];
        }
        if (!this.isConnected) {
            return null;
        }
        const formID = this.getAttribute('form');
        return formID
            ? this[PropertySymbol.ownerDocument].getElementById(formID)
            : null;
    }
    /**
     * Returns the associated label elements.
     *
     * @returns Label elements.
     */
    get labels() {
        return HTMLLabelElementUtility.getAssociatedLabelElements(this);
    }
    /**
     * Checks validity.
     *
     * @returns "true" if the field is valid.
     */
    checkValidity() {
        const valid = this.disabled ||
            this.type === 'reset' ||
            this.type === 'button' ||
            this[PropertySymbol.validity].valid;
        if (!valid) {
            this.dispatchEvent(new Event('invalid', { bubbles: true, cancelable: true }));
        }
        return valid;
    }
    /**
     * Reports validity.
     *
     * @returns Validity.
     */
    reportValidity() {
        return this.checkValidity();
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
     * @override
     */
    dispatchEvent(event) {
        if (event.type === 'click' &&
            event instanceof MouseEvent &&
            event.eventPhase === EventPhaseEnum.none &&
            this.disabled) {
            return false;
        }
        const returnValue = super.dispatchEvent(event);
        if (event.type === 'click' &&
            event instanceof MouseEvent &&
            (event.eventPhase === EventPhaseEnum.atTarget ||
                event.eventPhase === EventPhaseEnum.bubbling) &&
            this[PropertySymbol.isConnected]) {
            const form = this.form;
            if (!form) {
                return returnValue;
            }
            switch (this.type) {
                case 'submit':
                    form.requestSubmit(this);
                    break;
                case 'reset':
                    form.reset();
                    break;
            }
        }
        return returnValue;
    }
    /**
     * @override
     */
    [(_HTMLButtonElement_instances = new WeakSet(), _a = PropertySymbol.attributes, _b = PropertySymbol.validationMessage, _c = PropertySymbol.validity, PropertySymbol.connectToNode)](parentNode = null) {
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
_HTMLButtonElement_sanitizeType = function _HTMLButtonElement_sanitizeType(type) {
    type = (type && type.toLowerCase()) || 'submit';
    if (!BUTTON_TYPES.includes(type)) {
        type = 'submit';
    }
    return type;
};
export default HTMLButtonElement;
//# sourceMappingURL=HTMLButtonElement.js.map