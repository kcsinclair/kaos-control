import Event from '../../event/Event.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import BrowserWindow from '../../window/BrowserWindow.js';
import Document from '../document/Document.js';
import HTMLElement from '../html-element/HTMLElement.js';
import Node from '../node/Node.js';
import NamedNodeMap from '../../named-node-map/NamedNodeMap.js';
import CrossOriginBrowserWindow from '../../window/CrossOriginBrowserWindow.js';
import IBrowserFrame from '../../browser/types/IBrowserFrame.js';
import DOMTokenList from '../../dom-token-list/DOMTokenList.js';
/**
 * HTML Iframe Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLIFrameElement.
 */
export default class HTMLIFrameElement extends HTMLElement {
    #private;
    cloneNode: (deep?: boolean) => HTMLIFrameElement;
    onload: (event: Event) => void | null;
    onerror: (event: Event) => void | null;
    [PropertySymbol.attributes]: NamedNodeMap;
    [PropertySymbol.sandbox]: DOMTokenList;
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     */
    constructor(browserFrame: IBrowserFrame);
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
     * Returns allow.
     *
     * @returns Allow.
     */
    get allow(): string;
    /**
     * Sets allow.
     *
     * @param allow Allow.
     */
    set allow(allow: string);
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): string;
    /**
     * Sets height.
     *
     * @param height Height.
     */
    set height(height: string);
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width(): string;
    /**
     * Sets width.
     *
     * @param width Width.
     */
    set width(width: string);
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
     * Returns sandbox.
     *
     * @returns Sandbox.
     */
    get sandbox(): DOMTokenList;
    /**
     * Sets sandbox.
     */
    set sandbox(sandbox: string);
    /**
     * Returns srcdoc.
     *
     * @returns Srcdoc.
     */
    get srcdoc(): string;
    /**
     * Sets srcdoc.
     *
     * @param srcdoc Srcdoc.
     */
    set srcdoc(srcdoc: string);
    /**
     * Returns referrer policy.
     */
    get referrerPolicy(): string;
    /**
     * Sets referrer policy.
     *
     * @param referrerPolicy Referrer policy.
     */
    set referrerPolicy(referrerPolicy: string);
    /**
     * Returns content document.
     *
     * @returns Content document.
     */
    get contentDocument(): Document | null;
    /**
     * Returns content window.
     *
     * @returns Content window.
     */
    get contentWindow(): BrowserWindow | CrossOriginBrowserWindow | null;
    /**
     * @override
     */
    [PropertySymbol.connectToNode](parentNode?: Node): void;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): HTMLIFrameElement;
}
//# sourceMappingURL=HTMLIFrameElement.d.ts.map