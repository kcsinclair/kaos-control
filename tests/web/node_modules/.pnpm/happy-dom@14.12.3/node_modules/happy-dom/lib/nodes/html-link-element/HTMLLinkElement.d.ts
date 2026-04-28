import CSSStyleSheet from '../../css/CSSStyleSheet.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElement from '../html-element/HTMLElement.js';
import Event from '../../event/Event.js';
import ErrorEvent from '../../event/events/ErrorEvent.js';
import Node from '../../nodes/node/Node.js';
import DOMTokenList from '../../dom-token-list/DOMTokenList.js';
import NamedNodeMap from '../../named-node-map/NamedNodeMap.js';
import IBrowserFrame from '../../browser/types/IBrowserFrame.js';
/**
 * HTML Link Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLLinkElement.
 */
export default class HTMLLinkElement extends HTMLElement {
    #private;
    onerror: (event: ErrorEvent) => void;
    onload: (event: Event) => void;
    [PropertySymbol.attributes]: NamedNodeMap;
    [PropertySymbol.sheet]: CSSStyleSheet;
    [PropertySymbol.evaluateCSS]: boolean;
    [PropertySymbol.relList]: DOMTokenList;
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     */
    constructor(browserFrame: IBrowserFrame);
    /**
     * Returns sheet.
     */
    get sheet(): CSSStyleSheet;
    /**
     * Returns rel list.
     *
     * @returns Rel list.
     */
    get relList(): DOMTokenList;
    /**
     * Returns as.
     *
     * @returns As.
     */
    get as(): string;
    /**
     * Sets crossOrigin.
     *
     * @param crossOrigin CrossOrigin.
     */
    set as(as: string);
    /**
     * Returns crossOrigin.
     *
     * @returns CrossOrigin.
     */
    get crossOrigin(): string;
    /**
     * Sets crossOrigin.
     *
     * @param crossOrigin CrossOrigin.
     */
    set crossOrigin(crossOrigin: string);
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href(): string;
    /**
     * Sets href.
     *
     * @param href Href.
     */
    set href(href: string);
    /**
     * Returns hreflang.
     *
     * @returns Hreflang.
     */
    get hreflang(): string;
    /**
     * Sets hreflang.
     *
     * @param hreflang Hreflang.
     */
    set hreflang(hreflang: string);
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media(): string;
    /**
     * Sets media.
     *
     * @param media Media.
     */
    set media(media: string);
    /**
     * Returns referrerPolicy.
     *
     * @returns ReferrerPolicy.
     */
    get referrerPolicy(): string;
    /**
     * Sets referrerPolicy.
     *
     * @param referrerPolicy ReferrerPolicy.
     */
    set referrerPolicy(referrerPolicy: string);
    /**
     * Returns rel.
     *
     * @returns Rel.
     */
    get rel(): string;
    /**
     * Sets rel.
     *
     * @param rel Rel.
     */
    set rel(rel: string);
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type(): string;
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type: string);
    /**
     * @override
     */
    [PropertySymbol.connectToNode](parentNode?: Node): void;
}
//# sourceMappingURL=HTMLLinkElement.d.ts.map