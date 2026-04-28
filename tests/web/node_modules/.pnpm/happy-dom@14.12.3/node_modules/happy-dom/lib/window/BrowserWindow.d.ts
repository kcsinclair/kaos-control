/// <reference types="node" resolution-mode="require"/>
/// <reference types="node" resolution-mode="require"/>
/// <reference types="node" resolution-mode="require"/>
/// <reference types="node" resolution-mode="require"/>
/// <reference types="node" resolution-mode="require"/>
/// <reference types="node" resolution-mode="require"/>
/// <reference types="node" resolution-mode="require"/>
/// <reference types="node" resolution-mode="require"/>
import CustomElementRegistry from '../custom-element/CustomElementRegistry.js';
import * as PropertySymbol from '../PropertySymbol.js';
import DocumentImplementation from '../nodes/document/Document.js';
import HTMLDocumentImplementation from '../nodes/html-document/HTMLDocument.js';
import XMLDocumentImplementation from '../nodes/xml-document/XMLDocument.js';
import SVGDocumentImplementation from '../nodes/svg-document/SVGDocument.js';
import Node from '../nodes/node/Node.js';
import ShadowRoot from '../nodes/shadow-root/ShadowRoot.js';
import HTMLTemplateElement from '../nodes/html-template-element/HTMLTemplateElement.js';
import HTMLFormElementImplementation from '../nodes/html-form-element/HTMLFormElement.js';
import HTMLElement from '../nodes/html-element/HTMLElement.js';
import HTMLUnknownElement from '../nodes/html-unknown-element/HTMLUnknownElement.js';
import HTMLInputElement from '../nodes/html-input-element/HTMLInputElement.js';
import HTMLSelectElement from '../nodes/html-select-element/HTMLSelectElement.js';
import HTMLTextAreaElement from '../nodes/html-text-area-element/HTMLTextAreaElement.js';
import HTMLLinkElementImplementation from '../nodes/html-link-element/HTMLLinkElement.js';
import HTMLStyleElement from '../nodes/html-style-element/HTMLStyleElement.js';
import HTMLSlotElement from '../nodes/html-slot-element/HTMLSlotElement.js';
import HTMLLabelElement from '../nodes/html-label-element/HTMLLabelElement.js';
import HTMLMetaElement from '../nodes/html-meta-element/HTMLMetaElement.js';
import HTMLMediaElement from '../nodes/html-media-element/HTMLMediaElement.js';
import HTMLAudioElement from '../nodes/html-audio-element/HTMLAudioElement.js';
import HTMLVideoElement from '../nodes/html-video-element/HTMLVideoElement.js';
import HTMLBaseElement from '../nodes/html-base-element/HTMLBaseElement.js';
import HTMLIFrameElementImplementation from '../nodes/html-iframe-element/HTMLIFrameElement.js';
import HTMLDialogElement from '../nodes/html-dialog-element/HTMLDialogElement.js';
import SVGSVGElement from '../nodes/svg-element/SVGSVGElement.js';
import SVGElement from '../nodes/svg-element/SVGElement.js';
import SVGGraphicsElement from '../nodes/svg-element/SVGGraphicsElement.js';
import HTMLScriptElementImplementation from '../nodes/html-script-element/HTMLScriptElement.js';
import HTMLImageElement from '../nodes/html-image-element/HTMLImageElement.js';
import CharacterData from '../nodes/character-data/CharacterData.js';
import DocumentType from '../nodes/document-type/DocumentType.js';
import NodeIterator from '../tree-walker/NodeIterator.js';
import TreeWalker from '../tree-walker/TreeWalker.js';
import Event from '../event/Event.js';
import CustomEvent from '../event/events/CustomEvent.js';
import AnimationEvent from '../event/events/AnimationEvent.js';
import KeyboardEvent from '../event/events/KeyboardEvent.js';
import MessageEvent from '../event/events/MessageEvent.js';
import ProgressEvent from '../event/events/ProgressEvent.js';
import MediaQueryListEvent from '../event/events/MediaQueryListEvent.js';
import HashChangeEvent from '../event/events/HashChangeEvent.js';
import TouchEvent from '../event/events/TouchEvent.js';
import Touch from '../event/Touch.js';
import EventTarget from '../event/EventTarget.js';
import MessagePort from '../event/MessagePort.js';
import { URLSearchParams } from 'url';
import URL from '../url/URL.js';
import Location from '../location/Location.js';
import MutationObserver from '../mutation-observer/MutationObserver.js';
import MutationRecord from '../mutation-observer/MutationRecord.js';
import XMLSerializer from '../xml-serializer/XMLSerializer.js';
import ResizeObserver from '../resize-observer/ResizeObserver.js';
import Blob from '../file/Blob.js';
import File from '../file/File.js';
import DOMException from '../exception/DOMException.js';
import History from '../history/History.js';
import CSSStyleSheet from '../css/CSSStyleSheet.js';
import CSSStyleDeclaration from '../css/declaration/CSSStyleDeclaration.js';
import CSS from '../css/CSS.js';
import CSSUnitValue from '../css/CSSUnitValue.js';
import CSSRule from '../css/CSSRule.js';
import CSSContainerRule from '../css/rules/CSSContainerRule.js';
import CSSFontFaceRule from '../css/rules/CSSFontFaceRule.js';
import CSSKeyframeRule from '../css/rules/CSSKeyframeRule.js';
import CSSKeyframesRule from '../css/rules/CSSKeyframesRule.js';
import CSSMediaRule from '../css/rules/CSSMediaRule.js';
import CSSStyleRule from '../css/rules/CSSStyleRule.js';
import CSSSupportsRule from '../css/rules/CSSSupportsRule.js';
import MouseEvent from '../event/events/MouseEvent.js';
import PointerEvent from '../event/events/PointerEvent.js';
import FocusEvent from '../event/events/FocusEvent.js';
import WheelEvent from '../event/events/WheelEvent.js';
import DataTransfer from '../event/DataTransfer.js';
import DataTransferItem from '../event/DataTransferItem.js';
import DataTransferItemList from '../event/DataTransferItemList.js';
import InputEvent from '../event/events/InputEvent.js';
import UIEvent from '../event/UIEvent.js';
import ErrorEvent from '../event/events/ErrorEvent.js';
import StorageEvent from '../event/events/StorageEvent.js';
import SubmitEvent from '../event/events/SubmitEvent.js';
import Screen from '../screen/Screen.js';
import IRequestInit from '../fetch/types/IRequestInit.js';
import Storage from '../storage/Storage.js';
import HTMLCollection from '../nodes/element/HTMLCollection.js';
import HTMLFormControlsCollection from '../nodes/html-form-element/HTMLFormControlsCollection.js';
import NodeList from '../nodes/node/NodeList.js';
import MediaQueryList from '../match-media/MediaQueryList.js';
import Selection from '../selection/Selection.js';
import Navigator from '../navigator/Navigator.js';
import MimeType from '../navigator/MimeType.js';
import MimeTypeArray from '../navigator/MimeTypeArray.js';
import Plugin from '../navigator/Plugin.js';
import PluginArray from '../navigator/PluginArray.js';
import DOMRect from '../nodes/element/DOMRect.js';
import DOMRectReadOnly from '../nodes/element/DOMRectReadOnly.js';
import { Buffer } from 'buffer';
import { webcrypto } from 'crypto';
import XMLHttpRequestUpload from '../xml-http-request/XMLHttpRequestUpload.js';
import XMLHttpRequestEventTarget from '../xml-http-request/XMLHttpRequestEventTarget.js';
import Attr from '../nodes/attr/Attr.js';
import NamedNodeMap from '../named-node-map/NamedNodeMap.js';
import Element from '../nodes/element/Element.js';
import ProcessingInstruction from '../nodes/processing-instruction/ProcessingInstruction.js';
import FileList from '../nodes/html-input-element/FileList.js';
import Stream from 'stream';
import { ReadableStream } from 'stream/web';
import FormData from '../form-data/FormData.js';
import AbortController from '../fetch/AbortController.js';
import AbortSignal from '../fetch/AbortSignal.js';
import RadioNodeList from '../nodes/html-form-element/RadioNodeList.js';
import ValidityState from '../validity-state/ValidityState.js';
import Permissions from '../permissions/Permissions.js';
import PermissionStatus from '../permissions/PermissionStatus.js';
import Clipboard from '../clipboard/Clipboard.js';
import ClipboardItem from '../clipboard/ClipboardItem.js';
import ClipboardEvent from '../event/events/ClipboardEvent.js';
import Headers from '../fetch/Headers.js';
import XMLHttpRequestImplementation from '../xml-http-request/XMLHttpRequest.js';
import DocumentReadyStateManager from '../nodes/document/DocumentReadyStateManager.js';
import IBrowserFrame from '../browser/types/IBrowserFrame.js';
import HTMLAnchorElement from '../nodes/html-anchor-element/HTMLAnchorElement.js';
import HTMLButtonElement from '../nodes/html-button-element/HTMLButtonElement.js';
import HTMLOptionElement from '../nodes/html-option-element/HTMLOptionElement.js';
import HTMLOptGroupElement from '../nodes/html-opt-group-element/HTMLOptGroupElement.js';
import HTMLTimeElement from '../nodes/html-time-element/HTMLTimeElement.js';
import IResponseBody from '../fetch/types/IResponseBody.js';
import IResponseInit from '../fetch/types/IResponseInit.js';
import IRequestInfo from '../fetch/types/IRequestInfo.js';
import AudioImplementation from '../nodes/html-audio-element/Audio.js';
import ImageImplementation from '../nodes/html-image-element/Image.js';
import TextImplementation from '../nodes/text/Text.js';
import CommentImplementation from '../nodes/comment/Comment.js';
import DocumentFragmentImplementation from '../nodes/document-fragment/DocumentFragment.js';
import DOMParserImplementation from '../dom-parser/DOMParser.js';
import FileReaderImplementation from '../file/FileReader.js';
import RequestImplementation from '../fetch/Request.js';
import ResponseImplementation from '../fetch/Response.js';
import RangeImplementation from '../range/Range.js';
import INodeJSGlobal from './INodeJSGlobal.js';
import CrossOriginBrowserWindow from './CrossOriginBrowserWindow.js';
import Response from '../fetch/Response.js';
import AsyncTaskManager from '../async-task-manager/AsyncTaskManager.js';
/**
 * Browser window.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Window.
 */
export default class BrowserWindow extends EventTarget implements INodeJSGlobal {
    #private;
    readonly Node: typeof Node;
    readonly Attr: typeof Attr;
    readonly SVGSVGElement: typeof SVGSVGElement;
    readonly SVGElement: typeof SVGElement;
    readonly SVGGraphicsElement: typeof SVGGraphicsElement;
    readonly ShadowRoot: typeof ShadowRoot;
    readonly ProcessingInstruction: typeof ProcessingInstruction;
    readonly Element: typeof Element;
    readonly CharacterData: typeof CharacterData;
    readonly DocumentType: typeof DocumentType;
    readonly Document: new () => DocumentImplementation;
    readonly HTMLDocument: new () => HTMLDocumentImplementation;
    readonly XMLDocument: new () => XMLDocumentImplementation;
    readonly SVGDocument: new () => SVGDocumentImplementation;
    readonly Text: typeof TextImplementation;
    readonly Comment: typeof CommentImplementation;
    readonly HTMLAnchorElement: typeof HTMLAnchorElement;
    readonly HTMLButtonElement: typeof HTMLButtonElement;
    readonly HTMLOptGroupElement: typeof HTMLOptGroupElement;
    readonly HTMLOptionElement: typeof HTMLOptionElement;
    readonly HTMLElement: typeof HTMLElement;
    readonly HTMLUnknownElement: typeof HTMLUnknownElement;
    readonly HTMLTemplateElement: typeof HTMLTemplateElement;
    readonly HTMLInputElement: typeof HTMLInputElement;
    readonly HTMLSelectElement: typeof HTMLSelectElement;
    readonly HTMLTextAreaElement: typeof HTMLTextAreaElement;
    readonly HTMLImageElement: typeof HTMLImageElement;
    readonly HTMLStyleElement: typeof HTMLStyleElement;
    readonly HTMLLabelElement: typeof HTMLLabelElement;
    readonly HTMLSlotElement: typeof HTMLSlotElement;
    readonly HTMLMetaElement: typeof HTMLMetaElement;
    readonly HTMLMediaElement: typeof HTMLMediaElement;
    readonly HTMLAudioElement: typeof HTMLAudioElement;
    readonly HTMLVideoElement: typeof HTMLVideoElement;
    readonly HTMLBaseElement: typeof HTMLBaseElement;
    readonly HTMLDialogElement: typeof HTMLDialogElement;
    readonly HTMLScriptElement: typeof HTMLScriptElementImplementation;
    readonly HTMLLinkElement: typeof HTMLLinkElementImplementation;
    readonly HTMLIFrameElement: typeof HTMLIFrameElementImplementation;
    readonly HTMLFormElement: typeof HTMLFormElementImplementation;
    readonly HTMLTimeElement: typeof HTMLTimeElement;
    readonly HTMLHeadElement: typeof HTMLElement;
    readonly HTMLTitleElement: typeof HTMLElement;
    readonly HTMLBodyElement: typeof HTMLElement;
    readonly HTMLHeadingElement: typeof HTMLElement;
    readonly HTMLParagraphElement: typeof HTMLElement;
    readonly HTMLHRElement: typeof HTMLElement;
    readonly HTMLPreElement: typeof HTMLElement;
    readonly HTMLUListElement: typeof HTMLElement;
    readonly HTMLOListElement: typeof HTMLElement;
    readonly HTMLLElement: typeof HTMLElement;
    readonly HTMLMenuElement: typeof HTMLElement;
    readonly HTMLDListElement: typeof HTMLElement;
    readonly HTMLDivElement: typeof HTMLElement;
    readonly HTMLAreaElement: typeof HTMLElement;
    readonly HTMLBRElement: typeof HTMLElement;
    readonly HTMLCanvasElement: typeof HTMLElement;
    readonly HTMLDataElement: typeof HTMLElement;
    readonly HTMLDataListElement: typeof HTMLElement;
    readonly HTMLDetailsElement: typeof HTMLElement;
    readonly HTMLDirectoryElement: typeof HTMLElement;
    readonly HTMLFieldSetElement: typeof HTMLElement;
    readonly HTMLFontElement: typeof HTMLElement;
    readonly HTMLHtmlElement: typeof HTMLElement;
    readonly HTMLLegendElement: typeof HTMLElement;
    readonly HTMLMapElement: typeof HTMLElement;
    readonly HTMLMarqueeElement: typeof HTMLElement;
    readonly HTMLMeterElement: typeof HTMLElement;
    readonly HTMLModElement: typeof HTMLElement;
    readonly HTMLOutputElement: typeof HTMLElement;
    readonly HTMLPictureElement: typeof HTMLElement;
    readonly HTMLProgressElement: typeof HTMLElement;
    readonly HTMLQuoteElement: typeof HTMLElement;
    readonly HTMLSourceElement: typeof HTMLElement;
    readonly HTMLSpanElement: typeof HTMLElement;
    readonly HTMLTableCaptionElement: typeof HTMLElement;
    readonly HTMLTableCellElement: typeof HTMLElement;
    readonly HTMLTableColElement: typeof HTMLElement;
    readonly HTMLTableElement: typeof HTMLElement;
    readonly HTMLTableRowElement: typeof HTMLElement;
    readonly HTMLTableSectionElement: typeof HTMLElement;
    readonly HTMLFrameElement: typeof HTMLElement;
    readonly HTMLFrameSetElement: typeof HTMLElement;
    readonly HTMLEmbedElement: typeof HTMLElement;
    readonly HTMLObjectElement: typeof HTMLElement;
    readonly HTMLParamElement: typeof HTMLElement;
    readonly HTMLTrackElement: typeof HTMLElement;
    readonly Event: typeof Event;
    readonly UIEvent: typeof UIEvent;
    readonly CustomEvent: typeof CustomEvent;
    readonly AnimationEvent: typeof AnimationEvent;
    readonly KeyboardEvent: typeof KeyboardEvent;
    readonly MessageEvent: typeof MessageEvent;
    readonly MouseEvent: typeof MouseEvent;
    readonly PointerEvent: typeof PointerEvent;
    readonly FocusEvent: typeof FocusEvent;
    readonly WheelEvent: typeof WheelEvent;
    readonly InputEvent: typeof InputEvent;
    readonly ErrorEvent: typeof ErrorEvent;
    readonly StorageEvent: typeof StorageEvent;
    readonly SubmitEvent: typeof SubmitEvent;
    readonly ProgressEvent: typeof ProgressEvent;
    readonly MediaQueryListEvent: typeof MediaQueryListEvent;
    readonly HashChangeEvent: typeof HashChangeEvent;
    readonly ClipboardEvent: typeof ClipboardEvent;
    readonly TouchEvent: typeof TouchEvent;
    readonly Touch: typeof Touch;
    readonly AudioProcessingEvent: typeof Event;
    readonly BeforeInputEvent: typeof Event;
    readonly BeforeUnloadEvent: typeof Event;
    readonly BlobEvent: typeof Event;
    readonly CloseEvent: typeof Event;
    readonly CompositionEvent: typeof Event;
    readonly CSSFontFaceLoadEvent: typeof Event;
    readonly DeviceLightEvent: typeof Event;
    readonly DeviceMotionEvent: typeof Event;
    readonly DeviceOrientationEvent: typeof Event;
    readonly DeviceProximityEvent: typeof Event;
    readonly DOMTransactionEvent: typeof Event;
    readonly DragEvent: typeof Event;
    readonly EditingBeforeInputEvent: typeof Event;
    readonly FetchEvent: typeof Event;
    readonly GamepadEvent: typeof Event;
    readonly IDBVersionChangeEvent: typeof Event;
    readonly MediaStreamEvent: typeof Event;
    readonly MutationEvent: typeof Event;
    readonly OfflineAudioCompletionEvent: typeof Event;
    readonly OverconstrainedError: typeof Event;
    readonly PageTransitionEvent: typeof Event;
    readonly PaymentRequestUpdateEvent: typeof Event;
    readonly PopStateEvent: typeof Event;
    readonly RelatedEvent: typeof Event;
    readonly RTCDataChannelEvent: typeof Event;
    readonly RTCIdentityErrorEvent: typeof Event;
    readonly RTCIdentityEvent: typeof Event;
    readonly RTCPeerConnectionIceEvent: typeof Event;
    readonly SensorEvent: typeof Event;
    readonly SVGEvent: typeof Event;
    readonly SVGZoomEvent: typeof Event;
    readonly TimeEvent: typeof Event;
    readonly TrackEvent: typeof Event;
    readonly TransitionEvent: typeof Event;
    readonly UserProximityEvent: typeof Event;
    readonly WebGLContextEvent: typeof Event;
    readonly TextEvent: typeof Event;
    readonly NamedNodeMap: typeof NamedNodeMap;
    readonly NodeFilter: {
        FILTER_ACCEPT: number;
        FILTER_REJECT: number;
        FILTER_SKIP: number;
        SHOW_ALL: number;
        SHOW_ELEMENT: number;
        SHOW_ATTRIBUTE: number;
        SHOW_TEXT: number;
        SHOW_CDATA_SECTION: number;
        SHOW_ENTITY_REFERENCE: number;
        SHOW_ENTITY: number;
        SHOW_PROCESSING_INSTRUCTION: number;
        SHOW_COMMENT: number;
        SHOW_DOCUMENT: number;
        SHOW_DOCUMENT_TYPE: number;
        SHOW_DOCUMENT_FRAGMENT: number;
        SHOW_NOTATION: number;
    };
    readonly NodeIterator: typeof NodeIterator;
    readonly TreeWalker: typeof TreeWalker;
    readonly MutationObserver: typeof MutationObserver;
    readonly MutationRecord: typeof MutationRecord;
    readonly CSSStyleDeclaration: typeof CSSStyleDeclaration;
    readonly EventTarget: typeof EventTarget;
    readonly MessagePort: typeof MessagePort;
    readonly DataTransfer: typeof DataTransfer;
    readonly DataTransferItem: typeof DataTransferItem;
    readonly DataTransferItemList: typeof DataTransferItemList;
    readonly URL: typeof URL;
    readonly Location: typeof Location;
    readonly CustomElementRegistry: typeof CustomElementRegistry;
    readonly Window: typeof BrowserWindow;
    readonly XMLSerializer: typeof XMLSerializer;
    readonly ResizeObserver: typeof ResizeObserver;
    readonly CSSStyleSheet: typeof CSSStyleSheet;
    readonly Blob: typeof Blob;
    readonly File: typeof File;
    readonly DOMException: typeof DOMException;
    readonly History: typeof History;
    readonly Screen: typeof Screen;
    readonly Storage: typeof Storage;
    readonly URLSearchParams: typeof URLSearchParams;
    readonly HTMLCollection: typeof HTMLCollection;
    readonly HTMLFormControlsCollection: typeof HTMLFormControlsCollection;
    readonly NodeList: typeof NodeList;
    readonly CSSUnitValue: typeof CSSUnitValue;
    readonly CSSRule: typeof CSSRule;
    readonly CSSContainerRule: typeof CSSContainerRule;
    readonly CSSFontFaceRule: typeof CSSFontFaceRule;
    readonly CSSKeyframeRule: typeof CSSKeyframeRule;
    readonly CSSKeyframesRule: typeof CSSKeyframesRule;
    readonly CSSMediaRule: typeof CSSMediaRule;
    readonly CSSStyleRule: typeof CSSStyleRule;
    readonly CSSSupportsRule: typeof CSSSupportsRule;
    readonly Selection: typeof Selection;
    readonly Navigator: typeof Navigator;
    readonly MimeType: typeof MimeType;
    readonly MimeTypeArray: typeof MimeTypeArray;
    readonly Plugin: typeof Plugin;
    readonly PluginArray: typeof PluginArray;
    readonly FileList: typeof FileList;
    readonly DOMRect: typeof DOMRect;
    readonly DOMRectReadOnly: typeof DOMRectReadOnly;
    readonly RadioNodeList: typeof RadioNodeList;
    readonly ValidityState: typeof ValidityState;
    readonly Headers: typeof Headers;
    readonly Request: new (input: IRequestInfo, init?: IRequestInit) => RequestImplementation;
    readonly Response: {
        redirect: (url: string, status?: number) => ResponseImplementation;
        error: () => ResponseImplementation;
        json: (data: object, init?: IResponseInit) => ResponseImplementation;
        new (body?: IResponseBody, init?: IResponseInit): ResponseImplementation;
    };
    readonly XMLHttpRequestUpload: typeof XMLHttpRequestUpload;
    readonly XMLHttpRequestEventTarget: typeof XMLHttpRequestEventTarget;
    readonly ReadableStream: {
        new (underlyingSource: import("stream/web").UnderlyingByteSource, strategy?: import("stream/web").QueuingStrategy<Uint8Array>): ReadableStream<Uint8Array>;
        new <R = any>(underlyingSource?: import("stream/web").UnderlyingSource<R>, strategy?: import("stream/web").QueuingStrategy<R>): ReadableStream<R>;
        prototype: ReadableStream<any>;
    };
    readonly WritableStream: typeof Stream.Writable;
    readonly TransformStream: typeof Stream.Transform;
    readonly AbortController: typeof AbortController;
    readonly AbortSignal: typeof AbortSignal;
    readonly FormData: typeof FormData;
    readonly Permissions: typeof Permissions;
    readonly PermissionStatus: typeof PermissionStatus;
    readonly Clipboard: typeof Clipboard;
    readonly ClipboardItem: typeof ClipboardItem;
    readonly XMLHttpRequest: new () => XMLHttpRequestImplementation;
    readonly DOMParser: new () => DOMParserImplementation;
    readonly Range: new () => RangeImplementation;
    readonly FileReader: new () => FileReaderImplementation;
    readonly Image: typeof ImageImplementation;
    readonly DocumentFragment: typeof DocumentFragmentImplementation;
    readonly Audio: typeof AudioImplementation;
    onload: ((event: Event) => void) | null;
    onerror: ((event: ErrorEvent) => void) | null;
    readonly document: DocumentImplementation;
    readonly customElements: CustomElementRegistry;
    readonly self: BrowserWindow;
    readonly top: BrowserWindow;
    readonly parent: BrowserWindow;
    readonly window: BrowserWindow;
    readonly globalThis: BrowserWindow;
    readonly performance: typeof performance;
    readonly screenLeft: number;
    readonly screenTop: number;
    readonly screenX: number;
    readonly screenY: number;
    readonly crypto: typeof webcrypto;
    readonly closed = false;
    console: Console;
    name: string;
    Array: typeof Array;
    ArrayBuffer: typeof ArrayBuffer;
    Boolean: typeof Boolean;
    Buffer: typeof Buffer;
    DataView: typeof DataView;
    Date: typeof Date;
    Error: typeof Error;
    EvalError: typeof EvalError;
    Float32Array: typeof Float32Array;
    Float64Array: typeof Float64Array;
    Function: typeof Function;
    Infinity: typeof Infinity;
    Int16Array: typeof Int16Array;
    Int32Array: typeof Int32Array;
    Int8Array: typeof Int8Array;
    Intl: typeof Intl;
    JSON: typeof JSON;
    Map: MapConstructor;
    Math: typeof Math;
    NaN: typeof NaN;
    Number: typeof Number;
    Object: typeof Object;
    Promise: typeof Promise;
    RangeError: typeof RangeError;
    ReferenceError: typeof ReferenceError;
    RegExp: typeof RegExp;
    Set: SetConstructor;
    String: typeof String;
    Symbol: Function;
    SyntaxError: typeof SyntaxError;
    TypeError: typeof TypeError;
    URIError: typeof URIError;
    Uint16Array: typeof Uint16Array;
    Uint32Array: typeof Uint32Array;
    Uint8Array: typeof Uint8Array;
    Uint8ClampedArray: typeof Uint8ClampedArray;
    WeakMap: WeakMapConstructor;
    WeakSet: WeakSetConstructor;
    decodeURI: typeof decodeURI;
    decodeURIComponent: typeof decodeURIComponent;
    encodeURI: typeof encodeURI;
    encodeURIComponent: typeof encodeURIComponent;
    eval: typeof eval;
    /**
     * @deprecated
     */
    escape: (str: string) => string;
    global: typeof globalThis;
    isFinite: typeof isFinite;
    isNaN: typeof isNaN;
    parseFloat: typeof parseFloat;
    parseInt: typeof parseInt;
    undefined: typeof undefined;
    /**
     * @deprecated
     */
    unescape: (str: string) => string;
    gc: () => void;
    v8debug?: unknown;
    [PropertySymbol.captureEventListenerCount]: {
        [eventType: string]: number;
    };
    [PropertySymbol.mutationObservers]: MutationObserver[];
    readonly [PropertySymbol.readyStateManager]: DocumentReadyStateManager;
    [PropertySymbol.asyncTaskManager]: AsyncTaskManager | null;
    [PropertySymbol.location]: Location;
    [PropertySymbol.history]: History;
    [PropertySymbol.navigator]: Navigator;
    [PropertySymbol.screen]: Screen;
    [PropertySymbol.sessionStorage]: Storage;
    [PropertySymbol.localStorage]: Storage;
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     * @param [options] Options.
     * @param [options.url] URL.
     */
    constructor(browserFrame: IBrowserFrame, options?: {
        url?: string;
    });
    /**
     * Returns location.
     */
    get location(): Location;
    /**
     * Returns location.
     *
     * @param href Href.
     */
    set location(href: string);
    /**
     * Returns history.
     */
    get history(): History;
    /**
     * Returns navigator.
     */
    get navigator(): Navigator;
    /**
     * Returns screen.
     */
    get screen(): Screen;
    /**
     * Returns session storage.
     */
    get sessionStorage(): Storage;
    /**
     * Returns local storage.
     */
    get localStorage(): Storage;
    /**
     * Returns opener.
     *
     * @returns Opener.
     */
    get opener(): BrowserWindow | CrossOriginBrowserWindow | null;
    /**
     * The number of pixels that the document is currently scrolled horizontally.
     *
     * @returns Scroll X.
     */
    get scrollX(): number;
    /**
     * The read-only Window property pageXOffset is an alias for scrollX.
     *
     * @returns Scroll X.
     */
    get pageXOffset(): number;
    /**
     * The number of pixels that the document is currently scrolled vertically.
     *
     * @returns Scroll Y.
     */
    get scrollY(): number;
    /**
     * The read-only Window property pageYOffset is an alias for scrollY.
     *
     * @returns Scroll Y.
     */
    get pageYOffset(): number;
    /**
     * The CSS interface holds useful CSS-related methods.
     *
     * @returns CSS interface.
     */
    get CSS(): CSS;
    /**
     * Returns inner width.
     *
     * @returns Inner width.
     */
    get innerWidth(): number;
    /**
     * Sets inner width.
     *
     * @param value Inner width.
     */
    set innerWidth(value: number);
    /**
     * Returns inner height.
     *
     * @returns Inner height.
     */
    get innerHeight(): number;
    /**
     * Sets inner height.
     *
     * @param value Inner height.
     */
    set innerHeight(value: number);
    /**
     * Returns outer width.
     *
     * @returns Outer width.
     */
    get outerWidth(): number;
    /**
     * Sets outer width.
     *
     * @param value Outer width.
     */
    set outerWidth(value: number);
    /**
     * Returns outer height.
     *
     * @returns Outer height.
     */
    get outerHeight(): number;
    /**
     * Sets outer height.
     *
     * @param value Outer height.
     */
    set outerHeight(value: number);
    /**
     * Returns device pixel ratio.
     *
     * @returns Device pixel ratio.
     */
    get devicePixelRatio(): number;
    /**
     * Sets device pixel ratio.
     *
     * @param value Device pixel ratio.
     */
    set devicePixelRatio(value: number);
    /**
     * Returns an object containing the values of all CSS properties of an element.
     *
     * @param element Element.
     * @returns CSS style declaration.
     */
    getComputedStyle(element: Element): CSSStyleDeclaration;
    /**
     * Returns selection.
     *
     * @returns Selection.
     */
    getSelection(): Selection;
    /**
     * Scrolls to a particular set of coordinates.
     *
     * @param x X position or options object.
     * @param y Y position.
     */
    scroll(x: {
        top?: number;
        left?: number;
        behavior?: string;
    } | number, y?: number): void;
    /**
     * Scrolls to a particular set of coordinates.
     *
     * @param x X position or options object.
     * @param y Y position.
     */
    scrollTo(x: {
        top?: number;
        left?: number;
        behavior?: string;
    } | number, y?: number): void;
    /**
     * Shifts focus away from the window.
     */
    blur(): void;
    /**
     * Gives focus to the window.
     */
    focus(): void;
    /**
     * Loads a specified resource into a new or existing browsing context (that is, a tab, a window, or an iframe) under a specified name.
     *
     * @param [url] URL.
     * @param [target] Target.
     * @param [features] Window features.
     * @returns Window.
     */
    open(url?: string, target?: string, features?: string): BrowserWindow | CrossOriginBrowserWindow | null;
    /**
     * Closes the window.
     */
    close(): void;
    /**
     * Returns a new MediaQueryList object that can then be used to determine if the document matches the media query string.
     *
     * @param mediaQueryString A string specifying the media query to parse into a MediaQueryList.
     * @returns A new MediaQueryList.
     */
    matchMedia(mediaQueryString: string): MediaQueryList;
    /**
     * Sets a timer which executes a function once the timer expires.
     *
     * @param callback Function to be executed.
     * @param [delay=0] Delay in ms.
     * @param args Arguments passed to the callback function.
     * @returns Timeout ID.
     */
    setTimeout(callback: Function, delay?: number, ...args: unknown[]): NodeJS.Timeout;
    /**
     * Cancels a timeout previously established by calling setTimeout().
     *
     * @param id ID of the timeout.
     */
    clearTimeout(id: NodeJS.Timeout): void;
    /**
     * Calls a function with a fixed time delay between each call.
     *
     * @param callback Function to be executed.
     * @param [delay=0] Delay in ms.
     * @param args Arguments passed to the callback function.
     * @returns Interval ID.
     */
    setInterval(callback: Function, delay?: number, ...args: unknown[]): NodeJS.Timeout;
    /**
     * Cancels a timed repeating action which was previously established by a call to setInterval().
     *
     * @param id ID of the interval.
     */
    clearInterval(id: NodeJS.Timeout): void;
    /**
     * Mock animation frames with timeouts.
     *
     * @param callback Callback.
     * @returns ID.
     */
    requestAnimationFrame(callback: (timestamp: number) => void): NodeJS.Immediate;
    /**
     * Mock animation frames with timeouts.
     *
     * @param id ID.
     */
    cancelAnimationFrame(id: NodeJS.Immediate): void;
    /**
     * Queues a microtask to be executed at a safe time prior to control returning to the browser's event loop.
     *
     * @param callback Function to be executed.
     */
    queueMicrotask(callback: Function): void;
    /**
     * This method provides an easy, logical way to fetch resources asynchronously across the network.
     *
     * @param url URL.
     * @param [init] Init.
     * @returns Promise.
     */
    fetch(url: IRequestInfo, init?: IRequestInit): Promise<Response>;
    /**
     * Creates a Base64-encoded ASCII string from a binary string (i.e., a string in which each character in the string is treated as a byte of binary data).
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/btoa
     * @param data Binay data.
     * @returns Base64-encoded string.
     */
    btoa(data: unknown): string;
    /**
     * Decodes a string of data which has been encoded using Base64 encoding.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/atob
     * @see https://infra.spec.whatwg.org/#forgiving-base64-encode.
     * @see Https://html.spec.whatwg.org/multipage/webappapis.html#btoa.
     * @param data Binay string.
     * @returns An ASCII string containing decoded data from encodedData.
     */
    atob(data: unknown): string;
    /**
     * Safely enables cross-origin communication between Window objects; e.g., between a page and a pop-up that it spawned, or between a page and an iframe embedded within it.
     *
     * @param message Message.
     * @param [targetOrigin=*] Target origin.
     * @param _transfer Transfer. Not implemented.
     */
    postMessage(message: unknown, targetOrigin?: string, _transfer?: unknown[]): void;
    /**
     * Resizes the window.
     *
     * @param width Width.
     * @param height Height.
     */
    resizeTo(width: number, height: number): void;
    /**
     * Resizes the current window by a specified amount.
     *
     * @param width Width.
     * @param height Height.
     */
    resizeBy(width: number, height: number): void;
    /**
     * Setup of VM context.
     */
    protected [PropertySymbol.setupVMContext](): void;
    /**
     * Destroys the window.
     */
    [PropertySymbol.destroy](): void;
}
//# sourceMappingURL=BrowserWindow.d.ts.map