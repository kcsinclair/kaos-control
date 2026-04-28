import HTMLElement from '../html-element/HTMLElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import Event from '../../event/Event.cjs';
import ErrorEvent from '../../event/events/ErrorEvent.cjs';
import Node from '../../nodes/node/Node.cjs';
import NamedNodeMap from '../../named-node-map/NamedNodeMap.cjs';
import IBrowserFrame from '../../browser/types/IBrowserFrame.cjs';
/**
 * HTML Script Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLScriptElement.
 */
export default class HTMLScriptElement extends HTMLElement {
    #private;
    cloneNode: (deep?: boolean) => HTMLScriptElement;
    onerror: (event: ErrorEvent) => void;
    onload: (event: Event) => void;
    [PropertySymbol.attributes]: NamedNodeMap;
    [PropertySymbol.evaluateScript]: boolean;
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     */
    constructor(browserFrame: IBrowserFrame);
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
     * Returns source.
     *
     * @returns Source.
     */
    get src(): string;
    /**
     * Sets source.
     *
     * @param src Source.
     */
    set src(src: string);
    /**
     * Returns charset.
     *
     * @returns Charset.
     */
    get charset(): string;
    /**
     * Sets charset.
     *
     * @param charset Charset.
     */
    set charset(charset: string);
    /**
     * Returns lang.
     *
     * @returns Lang.
     */
    get lang(): string;
    /**
     * Sets lang.
     *
     * @param lang Lang.
     */
    set lang(lang: string);
    /**
     * Returns async.
     *
     * @returns Async.
     */
    get async(): boolean;
    /**
     * Sets async.
     *
     * @param async Async.
     */
    set async(async: boolean);
    /**
     * Returns defer.
     *
     * @returns Defer.
     */
    get defer(): boolean;
    /**
     * Sets defer.
     *
     * @param defer Defer.
     */
    set defer(defer: boolean);
    /**
     * Returns text.
     *
     * @returns Text.
     */
    get text(): string;
    /**
     * Sets text.
     *
     * @param text Text.
     */
    set text(text: string);
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): HTMLScriptElement;
    /**
     * @override
     */
    [PropertySymbol.connectToNode](parentNode?: Node): void;
}
//# sourceMappingURL=HTMLScriptElement.d.ts.map