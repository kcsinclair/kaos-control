import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Event from '../../event/Event.js';
import HTMLFormControlsCollection from './HTMLFormControlsCollection.js';
import Node from '../node/Node.js';
import HTMLInputElement from '../html-input-element/HTMLInputElement.js';
import HTMLTextAreaElement from '../html-text-area-element/HTMLTextAreaElement.js';
import HTMLSelectElement from '../html-select-element/HTMLSelectElement.js';
import HTMLButtonElement from '../html-button-element/HTMLButtonElement.js';
import IBrowserFrame from '../../browser/types/IBrowserFrame.js';
/**
 * HTML Form Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLFormElement.
 */
export default class HTMLFormElement extends HTMLElement {
    #private;
    cloneNode: (deep?: boolean) => HTMLFormElement;
    [PropertySymbol.elements]: HTMLFormControlsCollection;
    [PropertySymbol.length]: number;
    [PropertySymbol.formNode]: Node;
    onformdata: (event: Event) => void | null;
    onreset: (event: Event) => void | null;
    onsubmit: (event: Event) => void | null;
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     */
    constructor(browserFrame: IBrowserFrame);
    /**
     * Returns elements.
     *
     * @returns Elements.
     */
    get elements(): HTMLFormControlsCollection;
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length(): number;
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
     * Returns method.
     *
     * @returns Method.
     */
    get method(): string;
    /**
     * Sets method.
     *
     * @param method Method.
     */
    set method(method: string);
    /**
     * Returns target.
     *
     * @returns Target.
     */
    get target(): string;
    /**
     * Sets target.
     *
     * @param target Target.
     */
    set target(target: string);
    /**
     * Returns action.
     *
     * @returns Action.
     */
    get action(): string;
    /**
     * Sets action.
     *
     * @param action Action.
     */
    set action(action: string);
    /**
     * Returns encoding.
     *
     * @returns Encoding.
     */
    get encoding(): string;
    /**
     * Sets encoding.
     *
     * @param encoding Encoding.
     */
    set encoding(encoding: string);
    /**
     * Returns enctype.
     *
     * @returns Enctype.
     */
    get enctype(): string;
    /**
     * Sets enctype.
     *
     * @param enctype Enctype.
     */
    set enctype(enctype: string);
    /**
     * Returns autocomplete.
     *
     * @returns Autocomplete.
     */
    get autocomplete(): string;
    /**
     * Sets autocomplete.
     *
     * @param autocomplete Autocomplete.
     */
    set autocomplete(autocomplete: string);
    /**
     * Returns accept charset.
     *
     * @returns Accept charset.
     */
    get acceptCharset(): string;
    /**
     * Sets accept charset.
     *
     * @param acceptCharset Accept charset.
     */
    set acceptCharset(acceptCharset: string);
    /**
     * Returns no validate.
     *
     * @returns No validate.
     */
    get noValidate(): boolean;
    /**
     * Sets no validate.
     *
     * @param noValidate No validate.
     */
    set noValidate(noValidate: boolean);
    /**
     * Submits form. No submit event is raised. In particular, the form's "submit" event handler is not run.
     */
    submit(): void;
    /**
     * Submits form, reports validity and raises submit event.
     *
     * @param [submitter] Submitter.
     */
    requestSubmit(submitter?: HTMLInputElement | HTMLButtonElement): void;
    /**
     * Resets form.
     */
    reset(): void;
    /**
     * Checks validity.
     *
     * @returns "true" if validation does'nt fail.
     */
    checkValidity(): boolean;
    /**
     * Reports validity.
     *
     * @returns "true" if validation does'nt fail.
     */
    reportValidity(): boolean;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): HTMLFormElement;
    /**
     * Appends a form control item.
     *
     * @param node Node.
     * @param name Name
     */
    [PropertySymbol.appendFormControlItem](node: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | HTMLButtonElement, name: string): void;
    /**
     * Remove a form control item.
     *
     * @param node Node.
     * @param name Name.
     */
    [PropertySymbol.removeFormControlItem](node: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | HTMLButtonElement, name: string): void;
    /**
     * Returns "true" if the property name is valid.
     *
     * @param name Name.
     * @returns True if the property name is valid.
     */
    protected [PropertySymbol.isValidPropertyName](name: string): boolean;
}
//# sourceMappingURL=HTMLFormElement.d.ts.map