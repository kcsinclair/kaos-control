import Element from '../../../nodes/element/Element.cjs';
import CSSStyleDeclarationPropertyManager from '../property-manager/CSSStyleDeclarationPropertyManager.cjs';
/**
 * CSS Style Declaration utility
 */
export default class CSSStyleDeclarationElementStyle {
    private cache;
    private element;
    private computed;
    /**
     * Constructor.
     *
     * @param element Element.
     * @param [computed] Computed.
     */
    constructor(element: Element, computed?: boolean);
    /**
     * Returns element style properties.
     *
     * @returns Element style properties.
     */
    getElementStyle(): CSSStyleDeclarationPropertyManager;
    /**
     * Returns style sheets.
     *
     * @param element Element.
     * @returns Style sheets.
     */
    private getComputedElementStyle;
    /**
     * Applies CSS text to elements.
     *
     * @param options Options.
     * @param options.elements Elements.
     * @param options.cssRules CSS rules.
     * @param options.rootElement Root element.
     * @param [options.hostElement] Host element.
     */
    private parseCSSRules;
    /**
     * Parses CSS variables in a value.
     *
     * @param value Value.
     * @param cssVariables CSS variables.
     * @returns CSS value.
     */
    private parseCSSVariablesInValue;
    /**
     * Parses measurements in a value.
     *
     * @param options Options.
     * @param options.value Value.
     * @param options.rootFontSize Root font size.
     * @param options.parentFontSize Parent font size.
     * @param [options.parentSize] Parent width.
     * @returns CSS value.
     */
    private parseMeasurementsInValue;
}
//# sourceMappingURL=CSSStyleDeclarationElementStyle.d.ts.map