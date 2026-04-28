import Element from '../../nodes/element/Element.cjs';
import CSSRule from '../CSSRule.cjs';
/**
 * CSS Style Declaration.
 */
export default abstract class AbstractCSSStyleDeclaration {
    #private;
    readonly parentRule: CSSRule;
    /**
     * Constructor.
     *
     * @param [ownerElement] Computed style element.
     * @param [computed] Computed.
     */
    constructor(ownerElement?: Element, computed?: boolean);
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length(): number;
    /**
     * Returns the style decleration as a CSS text.
     *
     * @returns CSS text.
     */
    get cssText(): string;
    /**
     * Sets CSS text.
     *
     * @param cssText CSS text.
     */
    set cssText(cssText: string);
    /**
     * Returns item.
     *
     * @param index Index.
     * @returns Item.
     */
    item(index: number): string;
    /**
     * Set a property.
     *
     * @param name Property name.
     * @param value Value. Must not contain "!important" as that should be set using the priority parameter.
     * @param [priority] Can be "important", or an empty string.
     */
    setProperty(name: string, value: string, priority?: 'important' | '' | undefined): void;
    /**
     * Removes a property.
     *
     * @param name Property name in kebab case.
     * @param value Value. Must not contain "!important" as that should be set using the priority parameter.
     * @param [priority] Can be "important", or an empty string.
     */
    removeProperty(name: string): void;
    /**
     * Returns a property.
     *
     * @param name Property name in kebab case.
     * @returns Property value.
     */
    getPropertyValue(name: string): string;
    /**
     * Returns a property.
     *
     * @param name Property name in kebab case.
     * @returns "important" if set to be important.
     */
    getPropertyPriority(name: string): string;
}
//# sourceMappingURL=AbstractCSSStyleDeclaration.d.ts.map