import CSSStyleDeclaration from '../../css/declaration/CSSStyleDeclaration.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import Element from '../element/Element.cjs';
import SVGSVGElement from './SVGSVGElement.cjs';
import Event from '../../event/Event.cjs';
import NamedNodeMap from '../../named-node-map/NamedNodeMap.cjs';
import IDataset from '../element/IDataset.cjs';
/**
 * SVG Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/SVGElement.
 */
export default class SVGElement extends Element {
    #private;
    onabort: (event: Event) => void | null;
    onerror: (event: Event) => void | null;
    onload: (event: Event) => void | null;
    onresize: (event: Event) => void | null;
    onscroll: (event: Event) => void | null;
    onunload: (event: Event) => void | null;
    [PropertySymbol.attributes]: NamedNodeMap;
    [PropertySymbol.style]: CSSStyleDeclaration | null;
    /**
     * Returns viewport.
     *
     * @returns SVG rect.
     */
    get viewportElement(): SVGElement;
    /**
     * Returns current translate.
     *
     * @returns Element.
     */
    get ownerSVGElement(): SVGSVGElement;
    /**
     * Returns data set.
     *
     * @returns Data set.
     */
    get dataset(): IDataset;
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get style(): CSSStyleDeclaration;
    /**
     * Returns tab index.
     *
     * @returns Tab index.
     */
    get tabIndex(): number;
    /**
     * Returns tab index.
     *
     * @param tabIndex Tab index.
     */
    set tabIndex(tabIndex: number);
    /**
     * Triggers a blur event.
     */
    blur(): void;
    /**
     * Triggers a focus event.
     */
    focus(): void;
}
//# sourceMappingURL=SVGElement.d.ts.map