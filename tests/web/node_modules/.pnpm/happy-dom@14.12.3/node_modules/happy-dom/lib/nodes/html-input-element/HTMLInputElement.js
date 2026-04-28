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
var _HTMLInputElement_instances, _HTMLInputElement_selectionStart, _HTMLInputElement_selectionEnd, _HTMLInputElement_selectionDirection, _HTMLInputElement_isSelectionSupported, _HTMLInputElement_setChecked, _a, _b, _c, _d, _e, _f, _g, _h, _j;
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import ValidityState from '../../validity-state/ValidityState.js';
import DOMException from '../../exception/DOMException.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import Event from '../../event/Event.js';
import HTMLInputElementValueSanitizer from './HTMLInputElementValueSanitizer.js';
import HTMLInputElementSelectionModeEnum from './HTMLInputElementSelectionModeEnum.js';
import HTMLInputElementSelectionDirectionEnum from './HTMLInputElementSelectionDirectionEnum.js';
import HTMLInputElementValueStepping from './HTMLInputElementValueStepping.js';
import FileList from './FileList.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import HTMLInputElementDateUtility from './HTMLInputElementDateUtility.js';
import HTMLLabelElementUtility from '../html-label-element/HTMLLabelElementUtility.js';
import HTMLInputElementNamedNodeMap from './HTMLInputElementNamedNodeMap.js';
import { URL } from 'url';
import MouseEvent from '../../event/events/MouseEvent.js';
/**
 * HTML Input Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLInputElement.
 *
 * Used as reference for some of the logic (like selection range):
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/nodes/nodes/HTMLInputElement-impl.js (MIT licensed).
 */
class HTMLInputElement extends HTMLElement {
    constructor() {
        super(...arguments);
        _HTMLInputElement_instances.add(this);
        // Events
        this.oninput = null;
        this.oninvalid = null;
        this.onselectionchange = null;
        // Internal properties
        this[_a] = new HTMLInputElementNamedNodeMap(this);
        this[_b] = null;
        this[_c] = 0;
        this[_d] = 0;
        this[_e] = false;
        this[_f] = null;
        this[_g] = '';
        this[_h] = new ValidityState(this);
        this[_j] = new FileList();
        // Private properties
        _HTMLInputElement_selectionStart.set(this, null);
        _HTMLInputElement_selectionEnd.set(this, null);
        _HTMLInputElement_selectionDirection.set(this, HTMLInputElementSelectionDirectionEnum.none);
    }
    /**
     * Returns default checked.
     *
     * @returns Default checked.
     */
    get defaultChecked() {
        return this[PropertySymbol.defaultChecked];
    }
    /**
     * Sets default checked.
     *
     * @param defaultChecked Default checked.
     */
    set defaultChecked(defaultChecked) {
        this[PropertySymbol.defaultChecked] = defaultChecked;
    }
    /**
     * Returns files.
     *
     * @returns Files.
     */
    get files() {
        return this[PropertySymbol.files];
    }
    /**
     * Sets files.
     *
     * @param files Files.
     */
    set files(files) {
        this[PropertySymbol.files] = files;
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
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        return this[PropertySymbol.height];
    }
    /**
     * Sets height.
     *
     * @param height Height.
     */
    set height(height) {
        this[PropertySymbol.height] = height;
        this.setAttribute('height', String(height));
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        return this[PropertySymbol.width];
    }
    /**
     * Sets width.
     *
     * @param width Width.
     */
    set width(width) {
        this[PropertySymbol.width] = width;
        this.setAttribute('width', String(width));
    }
    /**
     * Returns size.
     *
     * @returns Size.
     */
    get size() {
        const size = this.getAttribute('size');
        if (size !== null) {
            return parseInt(size);
        }
        return 20;
    }
    /**
     * Sets size.
     *
     * @param size Size.
     */
    set size(size) {
        this.setAttribute('size', String(size));
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
     * Returns type.
     *
     * @returns Type. Defaults to "text".
     */
    get type() {
        return this.getAttribute('type') || 'text';
    }
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type) {
        this.setAttribute('type', type.toLowerCase());
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
     * Returns alt.
     *
     * @returns Alt.
     */
    get alt() {
        return this.getAttribute('alt') || '';
    }
    /**
     * Sets alt.
     *
     * @param alt Alt.
     */
    set alt(alt) {
        this.setAttribute('alt', alt);
    }
    /**
     * Returns min.
     *
     * @returns Min.
     */
    get min() {
        return this.getAttribute('min') || '';
    }
    /**
     * Sets min.
     *
     * @param min Min.
     */
    set min(min) {
        this.setAttribute('min', min);
    }
    /**
     * Returns max.
     *
     * @returns Max.
     */
    get max() {
        return this.getAttribute('max') || '';
    }
    /**
     * Sets max.
     *
     * @param max Max.
     */
    set max(max) {
        this.setAttribute('max', max);
    }
    /**
     * Returns pattern.
     *
     * @returns Pattern.
     */
    get pattern() {
        return this.getAttribute('pattern') || '';
    }
    /**
     * Sets pattern.
     *
     * @param pattern Pattern.
     */
    set pattern(pattern) {
        this.setAttribute('pattern', pattern);
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
     * Returns step.
     *
     * @returns Step.
     */
    get step() {
        return this.getAttribute('step') || '';
    }
    /**
     * Sets step.
     *
     * @param step Step.
     */
    set step(step) {
        this.setAttribute('step', step);
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
     * Returns accept.
     *
     * @returns Accept.
     */
    get accept() {
        return this.getAttribute('accept') || '';
    }
    /**
     * Sets accept.
     *
     * @param accept Accept.
     */
    set accept(accept) {
        this.setAttribute('accept', accept);
    }
    /**
     * Returns allowdirs.
     *
     * @returns Allowdirs.
     */
    get allowdirs() {
        return this.getAttribute('allowdirs') || '';
    }
    /**
     * Sets allowdirs.
     *
     * @param allowdirs Allowdirs.
     */
    set allowdirs(allowdirs) {
        this.setAttribute('allowdirs', allowdirs);
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
     * Returns src.
     *
     * @returns Src.
     */
    get src() {
        return this.getAttribute('src') || '';
    }
    /**
     * Sets src.
     *
     * @param src Src.
     */
    set src(src) {
        this.setAttribute('src', src);
    }
    /**
     * Returns defaultValue.
     *
     * @returns Defaultvalue.
     */
    get defaultValue() {
        return this.getAttribute('value') || '';
    }
    /**
     * Sets defaultValue.
     *
     * @param defaultValue Defaultvalue.
     */
    set defaultValue(defaultValue) {
        this.setAttribute('value', defaultValue);
    }
    /**
     * Returns read only.
     *
     * @returns Read only.
     */
    get readOnly() {
        return this.getAttribute('readonly') !== null;
    }
    /**
     * Sets read only.
     *
     * @param readOnly Read only.
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
     * Returns indeterminate.
     *
     * @returns Indeterminate.
     */
    get indeterminate() {
        return this.getAttribute('indeterminate') !== null;
    }
    /**
     * Sets indeterminate.
     *
     * @param indeterminate Indeterminate.
     */
    set indeterminate(indeterminate) {
        if (!indeterminate) {
            this.removeAttribute('indeterminate');
        }
        else {
            this.setAttribute('indeterminate', '');
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
     * Returns checked.
     *
     * @returns Checked.
     */
    get checked() {
        if (this[PropertySymbol.checked] !== null) {
            return this[PropertySymbol.checked];
        }
        return this.getAttribute('checked') !== null;
    }
    /**
     * Sets checked.
     *
     * @param checked Checked.
     */
    set checked(checked) {
        __classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_setChecked).call(this, checked);
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        switch (this.type) {
            case 'hidden':
            case 'submit':
            case 'image':
            case 'reset':
            case 'button':
                return this.getAttribute('value') || '';
            case 'checkbox':
            case 'radio':
                const attritube = this.getAttribute('value');
                return attritube !== null ? attritube : 'on';
            case 'file':
                return this[PropertySymbol.files].length > 0
                    ? '/fake/path/' + this[PropertySymbol.files][0].name
                    : '';
        }
        if (this[PropertySymbol.value] === null) {
            return this.getAttribute('value') || '';
        }
        return this[PropertySymbol.value];
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        // The value maybe not string, so we need to convert it to string
        value = String(value);
        switch (this.type) {
            case 'hidden':
            case 'submit':
            case 'image':
            case 'reset':
            case 'button':
            case 'checkbox':
            case 'radio':
                this.setAttribute('value', value);
                break;
            case 'file':
                if (value !== '') {
                    throw new DOMException('Input elements of type "file" may only programmatically set the value to empty string.', DOMExceptionNameEnum.invalidStateError);
                }
                break;
            default:
                const oldValue = this.value;
                this[PropertySymbol.value] = HTMLInputElementValueSanitizer.sanitize(this, value);
                if (oldValue !== this[PropertySymbol.value]) {
                    __classPrivateFieldSet(this, _HTMLInputElement_selectionStart, this[PropertySymbol.value].length, "f");
                    __classPrivateFieldSet(this, _HTMLInputElement_selectionEnd, this[PropertySymbol.value].length, "f");
                    __classPrivateFieldSet(this, _HTMLInputElement_selectionDirection, HTMLInputElementSelectionDirectionEnum.none, "f");
                }
                break;
        }
    }
    /**
     * Returns selection start.
     *
     * @returns Selection start.
     */
    get selectionStart() {
        if (!__classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_isSelectionSupported).call(this)) {
            return null;
        }
        if (__classPrivateFieldGet(this, _HTMLInputElement_selectionStart, "f") === null) {
            return this.value.length;
        }
        return __classPrivateFieldGet(this, _HTMLInputElement_selectionStart, "f");
    }
    /**
     * Sets selection start.
     *
     * @param start Start.
     */
    set selectionStart(start) {
        if (!__classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_isSelectionSupported).call(this)) {
            throw new DOMException(`The input element's type (${this.type}) does not support selection.`, DOMExceptionNameEnum.invalidStateError);
        }
        this.setSelectionRange(start, Math.max(start, this.selectionEnd), __classPrivateFieldGet(this, _HTMLInputElement_selectionDirection, "f"));
    }
    /**
     * Returns selection end.
     *
     * @returns Selection end.
     */
    get selectionEnd() {
        if (!__classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_isSelectionSupported).call(this)) {
            return null;
        }
        if (__classPrivateFieldGet(this, _HTMLInputElement_selectionEnd, "f") === null) {
            return this.value.length;
        }
        return __classPrivateFieldGet(this, _HTMLInputElement_selectionEnd, "f");
    }
    /**
     * Sets selection end.
     *
     * @param end End.
     */
    set selectionEnd(end) {
        if (!__classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_isSelectionSupported).call(this)) {
            throw new DOMException(`The input element's type (${this.type}) does not support selection.`, DOMExceptionNameEnum.invalidStateError);
        }
        this.setSelectionRange(this.selectionStart, end, __classPrivateFieldGet(this, _HTMLInputElement_selectionDirection, "f"));
    }
    /**
     * Returns selection direction.
     *
     * @returns Selection direction.
     */
    get selectionDirection() {
        if (!__classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_isSelectionSupported).call(this)) {
            return null;
        }
        return __classPrivateFieldGet(this, _HTMLInputElement_selectionDirection, "f");
    }
    /**
     * Sets selection direction.
     *
     * @param direction Direction.
     */
    set selectionDirection(direction) {
        if (!__classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_isSelectionSupported).call(this)) {
            throw new DOMException(`The input element's type (${this.type}) does not support selection.`, DOMExceptionNameEnum.invalidStateError);
        }
        this.setSelectionRange(__classPrivateFieldGet(this, _HTMLInputElement_selectionStart, "f"), __classPrivateFieldGet(this, _HTMLInputElement_selectionEnd, "f"), direction);
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
     * Returns value as Date.
     *
     * @returns Date.
     */
    get valueAsDate() {
        switch (this.type) {
            case 'date':
            case 'month':
                return isNaN(new Date(String(this.value)).getTime()) ? null : new Date(this.value);
            case 'week': {
                const d = HTMLInputElementDateUtility.isoWeekDate(this.value);
                return isNaN(d.getTime()) ? null : d;
            }
            case 'time': {
                const d = new Date(`1970-01-01T${this.value}Z`);
                return isNaN(d.getTime()) ? null : d;
            }
            default:
                return null;
        }
    }
    /**
     * Sets value from a Date.
     *
     * @param value Date.
     */
    set valueAsDate(value) {
        // Specs at https://html.spec.whatwg.org/multipage/input.html#dom-input-valueasdate
        if (!['date', 'month', 'time', 'week'].includes(this.type)) {
            throw new DOMException("Failed to set the 'valueAsDate' property on 'HTMLInputElement': This input element does not support Date values.", DOMExceptionNameEnum.invalidStateError);
        }
        if (typeof value !== 'object') {
            throw new TypeError("Failed to set the 'valueAsDate' property on 'HTMLInputElement': Failed to convert value to 'object'.");
        }
        else if (value && !(value instanceof Date)) {
            throw new TypeError("Failed to set the 'valueAsDate' property on 'HTMLInputElement': The provided value is not a Date.");
        }
        else if (value === null || isNaN(value.getTime())) {
            this.value = '';
            return;
        }
        switch (this.type) {
            case 'date':
                this.value = value.toISOString().split('T')[0];
                break;
            case 'month':
                this.value = value.toISOString().split('T')[0].slice(0, -3);
                break;
            case 'time':
                this.value = value.toISOString().split('T')[1].slice(0, 5);
                break;
            case 'week':
                this.value = HTMLInputElementDateUtility.dateIsoWeek(value);
                break;
        }
    }
    /**
     * Returns value as number.
     *
     * @returns Number.
     */
    get valueAsNumber() {
        const value = this.value;
        if (!this.type.match(/^(range|number|date|datetime-local|month|time|week)$/) || !value) {
            return NaN;
        }
        switch (this.type) {
            case 'number':
                return parseFloat(value);
            case 'range': {
                const number = parseFloat(value);
                const min = parseFloat(this.min) || 0;
                const max = parseFloat(this.max) || 100;
                if (isNaN(number)) {
                    return max < min ? min : (min + max) / 2;
                }
                else if (number < min) {
                    return min;
                }
                else if (number > max) {
                    return max;
                }
                return number;
            }
            case 'date':
                return new Date(value).getTime();
            case 'datetime-local':
                return new Date(value).getTime() - new Date(value).getTimezoneOffset() * 60000;
            case 'month':
                return (new Date(value).getUTCFullYear() - 1970) * 12 + new Date(value).getUTCMonth();
            case 'time':
                return (new Date('1970-01-01T' + value).getTime() - new Date('1970-01-01T00:00:00').getTime());
            case 'week': {
                // https://html.spec.whatwg.org/multipage/input.html#week-state-(type=week)
                const match = value.match(/^(\d{4})-W(\d{2})$/);
                if (!match) {
                    return NaN;
                }
                const d = new Date(Date.UTC(parseInt(match[1], 10), 0));
                const day = d.getUTCDay();
                const diff = ((day === 0 ? -6 : 1) - day) * 86400000 + parseInt(match[2], 10) * 604800000;
                return d.getTime() + diff;
            }
        }
    }
    /**
     * Sets value from a number.
     *
     * @param value number.
     */
    set valueAsNumber(value) {
        // Specs at https://html.spec.whatwg.org/multipage/input.html
        switch (this.type) {
            case 'number':
            case 'range':
                // We Rely on HTMLInputElementValueSanitizer
                this.value = Number(value).toString();
                break;
            case 'date':
            case 'datetime-local': {
                const d = new Date(Number(value));
                if (isNaN(d.getTime())) {
                    // Reset to default value
                    this.value = '';
                    break;
                }
                if (this.type == 'date') {
                    this.value = d.toISOString().slice(0, 10);
                }
                else {
                    this.value = d.toISOString().slice(0, -1);
                }
                break;
            }
            case 'month':
                if (!Number.isInteger(value) || value < 0) {
                    this.value = '';
                }
                else {
                    this.value = new Date(Date.UTC(1970, Number(value))).toISOString().slice(0, 7);
                }
                break;
            case 'time':
                if (!Number.isInteger(value) || value < 0) {
                    this.value = '';
                }
                else {
                    this.value = new Date(Number(value)).toISOString().slice(11, -1);
                }
                break;
            case 'week': {
                const d = new Date(Number(value));
                this.value = isNaN(d.getTime()) ? '' : HTMLInputElementDateUtility.dateIsoWeek(d);
                break;
            }
            default:
                throw new DOMException("Failed to set the 'valueAsNumber' property on 'HTMLInputElement': This input element does not support Number values.", DOMExceptionNameEnum.invalidStateError);
        }
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
     * Sets validation message.
     *
     * @param message Message.
     */
    setCustomValidity(message) {
        this[PropertySymbol.validationMessage] = String(message);
    }
    /**
     * Selects the text.
     */
    select() {
        if (!__classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_isSelectionSupported).call(this)) {
            return null;
        }
        __classPrivateFieldSet(this, _HTMLInputElement_selectionStart, 0, "f");
        __classPrivateFieldSet(this, _HTMLInputElement_selectionEnd, this.value.length, "f");
        __classPrivateFieldSet(this, _HTMLInputElement_selectionDirection, HTMLInputElementSelectionDirectionEnum.none, "f");
        this.dispatchEvent(new Event('select', { bubbles: true, cancelable: true }));
    }
    /**
     * Set selection range.
     *
     * @param start Start.
     * @param end End.
     * @param [direction="none"] Direction.
     */
    setSelectionRange(start, end, direction = 'none') {
        if (!__classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_isSelectionSupported).call(this)) {
            throw new DOMException(`The input element's type (${this.type}) does not support selection.`, DOMExceptionNameEnum.invalidStateError);
        }
        __classPrivateFieldSet(this, _HTMLInputElement_selectionEnd, Math.min(end, this.value.length), "f");
        __classPrivateFieldSet(this, _HTMLInputElement_selectionStart, Math.min(start, __classPrivateFieldGet(this, _HTMLInputElement_selectionEnd, "f")), "f");
        __classPrivateFieldSet(this, _HTMLInputElement_selectionDirection, direction === HTMLInputElementSelectionDirectionEnum.forward ||
            direction === HTMLInputElementSelectionDirectionEnum.backward
            ? direction
            : HTMLInputElementSelectionDirectionEnum.none, "f");
        this.dispatchEvent(new Event('select', { bubbles: true, cancelable: true }));
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
    setRangeText(replacement, start = null, end = null, selectionMode = HTMLInputElementSelectionModeEnum.preserve) {
        if (!__classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_isSelectionSupported).call(this)) {
            throw new DOMException(`The input element's type (${this.type}) does not support selection.`, DOMExceptionNameEnum.invalidStateError);
        }
        if (start === null) {
            start = __classPrivateFieldGet(this, _HTMLInputElement_selectionStart, "f");
        }
        if (end === null) {
            end = __classPrivateFieldGet(this, _HTMLInputElement_selectionEnd, "f");
        }
        if (start > end) {
            throw new DOMException('The index is not in the allowed range.', DOMExceptionNameEnum.invalidStateError);
        }
        start = Math.min(start, this.value.length);
        end = Math.min(end, this.value.length);
        const val = this.value;
        let selectionStart = __classPrivateFieldGet(this, _HTMLInputElement_selectionStart, "f");
        let selectionEnd = __classPrivateFieldGet(this, _HTMLInputElement_selectionEnd, "f");
        this.value = val.slice(0, start) + replacement + val.slice(end);
        const newEnd = start + this.value.length;
        switch (selectionMode) {
            case HTMLInputElementSelectionModeEnum.select:
                this.setSelectionRange(start, newEnd);
                break;
            case HTMLInputElementSelectionModeEnum.start:
                this.setSelectionRange(start, start);
                break;
            case HTMLInputElementSelectionModeEnum.end:
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
     * Checks validity.
     *
     * @returns "true" if the field is valid.
     */
    checkValidity() {
        const valid = this.disabled ||
            this.readOnly ||
            this.type === 'hidden' ||
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
     * @returns "true" if the field is valid.
     */
    reportValidity() {
        return this.checkValidity();
    }
    /**
     * Steps up.
     *
     * @param [increment] Increment.
     */
    stepUp(increment) {
        const newValue = HTMLInputElementValueStepping.step(this.type, this.value, 1, increment);
        if (newValue !== null) {
            this.value = newValue;
        }
    }
    /**
     * Steps down.
     *
     * @param [increment] Increment.
     */
    stepDown(increment) {
        const newValue = HTMLInputElementValueStepping.step(this.type, this.value, -1, increment);
        if (newValue !== null) {
            this.value = newValue;
        }
    }
    /**
     * @override
     */
    [(_HTMLInputElement_selectionStart = new WeakMap(), _HTMLInputElement_selectionEnd = new WeakMap(), _HTMLInputElement_selectionDirection = new WeakMap(), _HTMLInputElement_instances = new WeakSet(), _a = PropertySymbol.attributes, _b = PropertySymbol.value, _c = PropertySymbol.height, _d = PropertySymbol.width, _e = PropertySymbol.defaultChecked, _f = PropertySymbol.checked, _g = PropertySymbol.validationMessage, _h = PropertySymbol.validity, _j = PropertySymbol.files, PropertySymbol.cloneNode)](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone.formAction = this.formAction;
        clone.formMethod = this.formMethod;
        clone[PropertySymbol.value] = this[PropertySymbol.value];
        clone[PropertySymbol.height] = this[PropertySymbol.height];
        clone[PropertySymbol.width] = this[PropertySymbol.width];
        clone[PropertySymbol.defaultChecked] = this[PropertySymbol.defaultChecked];
        clone[PropertySymbol.files] = this[PropertySymbol.files].slice();
        __classPrivateFieldSet(clone, _HTMLInputElement_selectionStart, __classPrivateFieldGet(this, _HTMLInputElement_selectionStart, "f"), "f");
        __classPrivateFieldSet(clone, _HTMLInputElement_selectionEnd, __classPrivateFieldGet(this, _HTMLInputElement_selectionEnd, "f"), "f");
        __classPrivateFieldSet(clone, _HTMLInputElement_selectionDirection, __classPrivateFieldGet(this, _HTMLInputElement_selectionDirection, "f"), "f");
        return clone;
    }
    /**
     * @override
     */
    dispatchEvent(event) {
        // Do nothing if the input element is disabled and the event is a click event.
        if (event.type === 'click' &&
            event instanceof MouseEvent &&
            event.eventPhase === EventPhaseEnum.none &&
            this.disabled) {
            return false;
        }
        let previousCheckedValue = null;
        // The checkbox or radio button has to be checked before the click event is dispatched, so that event listeners can check the checked value.
        // However, the value has to be restored if preventDefault() is called on the click event.
        if ((event.eventPhase === EventPhaseEnum.atTarget ||
            event.eventPhase === EventPhaseEnum.bubbling) &&
            event.type === 'click' &&
            event instanceof MouseEvent) {
            const inputType = this.type;
            if (inputType === 'checkbox' || inputType === 'radio') {
                previousCheckedValue = this.checked;
                __classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_setChecked).call(this, inputType === 'checkbox' ? !previousCheckedValue : true);
            }
        }
        const returnValue = super.dispatchEvent(event);
        if (!event.defaultPrevented &&
            (event.eventPhase === EventPhaseEnum.atTarget ||
                event.eventPhase === EventPhaseEnum.bubbling) &&
            event.type === 'click' &&
            event instanceof MouseEvent &&
            this[PropertySymbol.isConnected]) {
            const inputType = this.type;
            if (!this.readOnly || inputType === 'checkbox' || inputType === 'radio') {
                if (inputType === 'checkbox' || inputType === 'radio') {
                    this.dispatchEvent(new Event('input', { bubbles: true, cancelable: true }));
                    this.dispatchEvent(new Event('change', { bubbles: true, cancelable: true }));
                }
                else if (inputType === 'submit') {
                    const form = this.form;
                    if (form) {
                        form.requestSubmit(this);
                    }
                }
                else if (inputType === 'reset' && this[PropertySymbol.isConnected]) {
                    const form = this.form;
                    if (form) {
                        form.reset();
                    }
                }
            }
        }
        // Restore checked state if preventDefault() is triggered on the click event.
        if (event.defaultPrevented &&
            (event.eventPhase === EventPhaseEnum.atTarget ||
                event.eventPhase === EventPhaseEnum.bubbling) &&
            event.type === 'click' &&
            event instanceof MouseEvent &&
            previousCheckedValue !== null) {
            const inputType = this.type;
            if (inputType === 'checkbox' || inputType === 'radio') {
                __classPrivateFieldGet(this, _HTMLInputElement_instances, "m", _HTMLInputElement_setChecked).call(this, previousCheckedValue);
            }
        }
        return returnValue;
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
_HTMLInputElement_isSelectionSupported = function _HTMLInputElement_isSelectionSupported() {
    const inputType = this.type;
    return (inputType === 'text' ||
        inputType === 'search' ||
        inputType === 'url' ||
        inputType === 'tel' ||
        inputType === 'password');
}, _HTMLInputElement_setChecked = function _HTMLInputElement_setChecked(checked) {
    this[PropertySymbol.checked] = checked;
    if (checked && this.type === 'radio' && this.name) {
        const root = ((this[PropertySymbol.formNode] || this.getRootNode()));
        const radioButtons = root.querySelectorAll(`input[type="radio"][name="${this.name}"]`);
        for (const radioButton of radioButtons) {
            if (radioButton !== this) {
                radioButton[PropertySymbol.checked] = false;
            }
        }
    }
};
export default HTMLInputElement;
//# sourceMappingURL=HTMLInputElement.js.map