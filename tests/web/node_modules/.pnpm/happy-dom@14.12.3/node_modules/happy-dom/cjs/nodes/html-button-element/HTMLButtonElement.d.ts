import Event from '../../event/Event.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import NamedNodeMap from '../../named-node-map/NamedNodeMap.cjs';
import ValidityState from '../../validity-state/ValidityState.cjs';
import HTMLElement from '../html-element/HTMLElement.cjs';
import HTMLFormElement from '../html-form-element/HTMLFormElement.cjs';
import HTMLLabelElement from '../html-label-element/HTMLLabelElement.cjs';
import Node from '../node/Node.cjs';
import NodeList from '../node/NodeList.cjs';
/**
 * HTML Button Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLButtonElement.
 */
export default class HTMLButtonElement extends HTMLElement {
    #private;
    [PropertySymbol.attributes]: NamedNodeMap;
    [PropertySymbol.validationMessage]: string;
    [PropertySymbol.validity]: ValidityState;
    /**
     * Returns validation message.
     *
     * @returns Validation message.
     */
    get validationMessage(): string;
    /**
     * Returns validity.
     *
     * @returns Validity.
     */
    get validity(): ValidityState;
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name(): string;
    /**
     * Sets name.
     *
     * @param name Name.
     */
    set name(name: string);
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value(): string;
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value: string);
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled(): boolean;
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled: boolean);
    /**
     * Returns type
     *
     * @returns Type
     */
    get type(): string;
    /**
     * Sets type
     *
     * @param v Type
     */
    set type(v: string);
    /**
     * Returns form action.
     *
     * @returns Form action.
     */
    get formAction(): string;
    /**
     * Sets form action.
     *
     * @param formAction Form action.
     */
    set formAction(formAction: string);
    /**
     * Returns form enctype.
     *
     * @returns Form enctype.
     */
    get formEnctype(): string;
    /**
     * Sets form enctype.
     *
     * @param formEnctype Form enctype.
     */
    set formEnctype(formEnctype: string);
    /**
     * Returns form method.
     *
     * @returns Form method.
     */
    get formMethod(): string;
    /**
     * Sets form method.
     *
     * @param formMethod Form method.
     */
    set formMethod(formMethod: string);
    /**
     * Returns no validate.
     *
     * @returns No validate.
     */
    get formNoValidate(): boolean;
    /**
     * Sets no validate.
     *
     * @param formNoValidate No validate.
     */
    set formNoValidate(formNoValidate: boolean);
    /**
     * Returns form target.
     *
     * @returns Form target.
     */
    get formTarget(): string;
    /**
     * Sets form target.
     *
     * @param formTarget Form target.
     */
    set formTarget(formTarget: string);
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form(): HTMLFormElement | null;
    /**
     * Returns the associated label elements.
     *
     * @returns Label elements.
     */
    get labels(): NodeList<HTMLLabelElement>;
    /**
     * Checks validity.
     *
     * @returns "true" if the field is valid.
     */
    checkValidity(): boolean;
    /**
     * Reports validity.
     *
     * @returns Validity.
     */
    reportValidity(): boolean;
    /**
     * Sets validation message.
     *
     * @param message Message.
     */
    setCustomValidity(message: string): void;
    /**
     * @override
     */
    dispatchEvent(event: Event): boolean;
    /**
     * @override
     */
    [PropertySymbol.connectToNode](parentNode?: Node): void;
}
//# sourceMappingURL=HTMLButtonElement.d.ts.map