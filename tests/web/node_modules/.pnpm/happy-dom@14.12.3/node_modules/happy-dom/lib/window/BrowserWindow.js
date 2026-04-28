var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _BrowserWindow_instances, _a, _BrowserWindow_browserFrame, _BrowserWindow_innerWidth, _BrowserWindow_innerHeight, _BrowserWindow_outerWidth, _BrowserWindow_outerHeight, _BrowserWindow_devicePixelRatio, _BrowserWindow_zeroTimeouts, _BrowserWindow_bindToThisScope, _b, _c, _d, _e;
import CustomElementRegistry from '../custom-element/CustomElementRegistry.js';
import * as PropertySymbol from '../PropertySymbol.js';
import DocumentImplementation from '../nodes/document/Document.js';
import HTMLDocumentImplementation from '../nodes/html-document/HTMLDocument.js';
import XMLDocumentImplementation from '../nodes/xml-document/XMLDocument.js';
import SVGDocumentImplementation from '../nodes/svg-document/SVGDocument.js';
import Node from '../nodes/node/Node.js';
import NodeFilter from '../tree-walker/NodeFilter.js';
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
import Storage from '../storage/Storage.js';
import StorageFactory from '../storage/StorageFactory.js';
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
import Fetch from '../fetch/Fetch.js';
import DOMRect from '../nodes/element/DOMRect.js';
import DOMRectReadOnly from '../nodes/element/DOMRectReadOnly.js';
import VMGlobalPropertyScript from './VMGlobalPropertyScript.js';
import VM from 'vm';
import { Buffer } from 'buffer';
import { webcrypto } from 'crypto';
import XMLHttpRequestUpload from '../xml-http-request/XMLHttpRequestUpload.js';
import XMLHttpRequestEventTarget from '../xml-http-request/XMLHttpRequestEventTarget.js';
import Base64 from '../base64/Base64.js';
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
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import RadioNodeList from '../nodes/html-form-element/RadioNodeList.js';
import ValidityState from '../validity-state/ValidityState.js';
import WindowErrorUtility from './WindowErrorUtility.js';
import Permissions from '../permissions/Permissions.js';
import PermissionStatus from '../permissions/PermissionStatus.js';
import Clipboard from '../clipboard/Clipboard.js';
import ClipboardItem from '../clipboard/ClipboardItem.js';
import ClipboardEvent from '../event/events/ClipboardEvent.js';
import Headers from '../fetch/Headers.js';
import XMLHttpRequestImplementation from '../xml-http-request/XMLHttpRequest.js';
import WindowBrowserSettingsReader from './WindowBrowserSettingsReader.js';
import DocumentReadyStateManager from '../nodes/document/DocumentReadyStateManager.js';
import DocumentReadyStateEnum from '../nodes/document/DocumentReadyStateEnum.js';
import HTMLAnchorElement from '../nodes/html-anchor-element/HTMLAnchorElement.js';
import HTMLButtonElement from '../nodes/html-button-element/HTMLButtonElement.js';
import HTMLOptionElement from '../nodes/html-option-element/HTMLOptionElement.js';
import HTMLOptGroupElement from '../nodes/html-opt-group-element/HTMLOptGroupElement.js';
import HTMLTimeElement from '../nodes/html-time-element/HTMLTimeElement.js';
import WindowPageOpenUtility from './WindowPageOpenUtility.js';
import BrowserErrorCaptureEnum from '../browser/enums/BrowserErrorCaptureEnum.js';
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
const TIMER = {
    setTimeout: globalThis.setTimeout.bind(globalThis),
    clearTimeout: globalThis.clearTimeout.bind(globalThis),
    setInterval: globalThis.setInterval.bind(globalThis),
    clearInterval: globalThis.clearInterval.bind(globalThis),
    queueMicrotask: globalThis.queueMicrotask.bind(globalThis),
    setImmediate: globalThis.setImmediate.bind(globalThis),
    clearImmediate: globalThis.clearImmediate.bind(globalThis)
};
const IS_NODE_JS_TIMEOUT_ENVIRONMENT = setTimeout.toString().includes('new Timeout');
/**
 * Zero Timeout.
 */
class Timeout {
    /**
     * Constructor.
     * @param callback Callback.
     */
    constructor(callback) {
        this.callback = callback;
    }
}
/**
 * Browser window.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Window.
 */
class BrowserWindow extends EventTarget {
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     * @param [options] Options.
     * @param [options.url] URL.
     */
    constructor(browserFrame, options) {
        var _f;
        super();
        _BrowserWindow_instances.add(this);
        // Nodes
        this.Node = Node;
        this.Attr = Attr;
        this.SVGSVGElement = SVGSVGElement;
        this.SVGElement = SVGElement;
        this.SVGGraphicsElement = SVGGraphicsElement;
        this.ShadowRoot = ShadowRoot;
        this.ProcessingInstruction = ProcessingInstruction;
        this.Element = Element;
        this.CharacterData = CharacterData;
        this.DocumentType = DocumentType;
        // Element classes
        this.HTMLAnchorElement = HTMLAnchorElement;
        this.HTMLButtonElement = HTMLButtonElement;
        this.HTMLOptGroupElement = HTMLOptGroupElement;
        this.HTMLOptionElement = HTMLOptionElement;
        this.HTMLElement = HTMLElement;
        this.HTMLUnknownElement = HTMLUnknownElement;
        this.HTMLTemplateElement = HTMLTemplateElement;
        this.HTMLInputElement = HTMLInputElement;
        this.HTMLSelectElement = HTMLSelectElement;
        this.HTMLTextAreaElement = HTMLTextAreaElement;
        this.HTMLImageElement = HTMLImageElement;
        this.HTMLStyleElement = HTMLStyleElement;
        this.HTMLLabelElement = HTMLLabelElement;
        this.HTMLSlotElement = HTMLSlotElement;
        this.HTMLMetaElement = HTMLMetaElement;
        this.HTMLMediaElement = HTMLMediaElement;
        this.HTMLAudioElement = HTMLAudioElement;
        this.HTMLVideoElement = HTMLVideoElement;
        this.HTMLBaseElement = HTMLBaseElement;
        this.HTMLDialogElement = HTMLDialogElement;
        this.HTMLTimeElement = HTMLTimeElement;
        // Non-implemented element classes
        this.HTMLHeadElement = HTMLElement;
        this.HTMLTitleElement = HTMLElement;
        this.HTMLBodyElement = HTMLElement;
        this.HTMLHeadingElement = HTMLElement;
        this.HTMLParagraphElement = HTMLElement;
        this.HTMLHRElement = HTMLElement;
        this.HTMLPreElement = HTMLElement;
        this.HTMLUListElement = HTMLElement;
        this.HTMLOListElement = HTMLElement;
        this.HTMLLElement = HTMLElement;
        this.HTMLMenuElement = HTMLElement;
        this.HTMLDListElement = HTMLElement;
        this.HTMLDivElement = HTMLElement;
        this.HTMLAreaElement = HTMLElement;
        this.HTMLBRElement = HTMLElement;
        this.HTMLCanvasElement = HTMLElement;
        this.HTMLDataElement = HTMLElement;
        this.HTMLDataListElement = HTMLElement;
        this.HTMLDetailsElement = HTMLElement;
        this.HTMLDirectoryElement = HTMLElement;
        this.HTMLFieldSetElement = HTMLElement;
        this.HTMLFontElement = HTMLElement;
        this.HTMLHtmlElement = HTMLElement;
        this.HTMLLegendElement = HTMLElement;
        this.HTMLMapElement = HTMLElement;
        this.HTMLMarqueeElement = HTMLElement;
        this.HTMLMeterElement = HTMLElement;
        this.HTMLModElement = HTMLElement;
        this.HTMLOutputElement = HTMLElement;
        this.HTMLPictureElement = HTMLElement;
        this.HTMLProgressElement = HTMLElement;
        this.HTMLQuoteElement = HTMLElement;
        this.HTMLSourceElement = HTMLElement;
        this.HTMLSpanElement = HTMLElement;
        this.HTMLTableCaptionElement = HTMLElement;
        this.HTMLTableCellElement = HTMLElement;
        this.HTMLTableColElement = HTMLElement;
        this.HTMLTableElement = HTMLElement;
        this.HTMLTableRowElement = HTMLElement;
        this.HTMLTableSectionElement = HTMLElement;
        this.HTMLFrameElement = HTMLElement;
        this.HTMLFrameSetElement = HTMLElement;
        this.HTMLEmbedElement = HTMLElement;
        this.HTMLObjectElement = HTMLElement;
        this.HTMLParamElement = HTMLElement;
        this.HTMLTrackElement = HTMLElement;
        // Event classes
        this.Event = Event;
        this.UIEvent = UIEvent;
        this.CustomEvent = CustomEvent;
        this.AnimationEvent = AnimationEvent;
        this.KeyboardEvent = KeyboardEvent;
        this.MessageEvent = MessageEvent;
        this.MouseEvent = MouseEvent;
        this.PointerEvent = PointerEvent;
        this.FocusEvent = FocusEvent;
        this.WheelEvent = WheelEvent;
        this.InputEvent = InputEvent;
        this.ErrorEvent = ErrorEvent;
        this.StorageEvent = StorageEvent;
        this.SubmitEvent = SubmitEvent;
        this.ProgressEvent = ProgressEvent;
        this.MediaQueryListEvent = MediaQueryListEvent;
        this.HashChangeEvent = HashChangeEvent;
        this.ClipboardEvent = ClipboardEvent;
        this.TouchEvent = TouchEvent;
        this.Touch = Touch;
        // Non-implemented event classes
        this.AudioProcessingEvent = Event;
        this.BeforeInputEvent = Event;
        this.BeforeUnloadEvent = Event;
        this.BlobEvent = Event;
        this.CloseEvent = Event;
        this.CompositionEvent = Event;
        this.CSSFontFaceLoadEvent = Event;
        this.DeviceLightEvent = Event;
        this.DeviceMotionEvent = Event;
        this.DeviceOrientationEvent = Event;
        this.DeviceProximityEvent = Event;
        this.DOMTransactionEvent = Event;
        this.DragEvent = Event;
        this.EditingBeforeInputEvent = Event;
        this.FetchEvent = Event;
        this.GamepadEvent = Event;
        this.IDBVersionChangeEvent = Event;
        this.MediaStreamEvent = Event;
        this.MutationEvent = Event;
        this.OfflineAudioCompletionEvent = Event;
        this.OverconstrainedError = Event;
        this.PageTransitionEvent = Event;
        this.PaymentRequestUpdateEvent = Event;
        this.PopStateEvent = Event;
        this.RelatedEvent = Event;
        this.RTCDataChannelEvent = Event;
        this.RTCIdentityErrorEvent = Event;
        this.RTCIdentityEvent = Event;
        this.RTCPeerConnectionIceEvent = Event;
        this.SensorEvent = Event;
        this.SVGEvent = Event;
        this.SVGZoomEvent = Event;
        this.TimeEvent = Event;
        this.TrackEvent = Event;
        this.TransitionEvent = Event;
        this.UserProximityEvent = Event;
        this.WebGLContextEvent = Event;
        this.TextEvent = Event;
        // Other classes
        this.NamedNodeMap = NamedNodeMap;
        this.NodeFilter = NodeFilter;
        this.NodeIterator = NodeIterator;
        this.TreeWalker = TreeWalker;
        this.MutationObserver = MutationObserver;
        this.MutationRecord = MutationRecord;
        this.CSSStyleDeclaration = CSSStyleDeclaration;
        this.EventTarget = EventTarget;
        this.MessagePort = MessagePort;
        this.DataTransfer = DataTransfer;
        this.DataTransferItem = DataTransferItem;
        this.DataTransferItemList = DataTransferItemList;
        this.URL = URL;
        this.Location = Location;
        this.CustomElementRegistry = CustomElementRegistry;
        this.Window = this.constructor;
        this.XMLSerializer = XMLSerializer;
        this.ResizeObserver = ResizeObserver;
        this.CSSStyleSheet = CSSStyleSheet;
        this.Blob = Blob;
        this.File = File;
        this.DOMException = DOMException;
        this.History = History;
        this.Screen = Screen;
        this.Storage = Storage;
        this.URLSearchParams = URLSearchParams;
        this.HTMLCollection = HTMLCollection;
        this.HTMLFormControlsCollection = HTMLFormControlsCollection;
        this.NodeList = NodeList;
        this.CSSUnitValue = CSSUnitValue;
        this.CSSRule = CSSRule;
        this.CSSContainerRule = CSSContainerRule;
        this.CSSFontFaceRule = CSSFontFaceRule;
        this.CSSKeyframeRule = CSSKeyframeRule;
        this.CSSKeyframesRule = CSSKeyframesRule;
        this.CSSMediaRule = CSSMediaRule;
        this.CSSStyleRule = CSSStyleRule;
        this.CSSSupportsRule = CSSSupportsRule;
        this.Selection = Selection;
        this.Navigator = Navigator;
        this.MimeType = MimeType;
        this.MimeTypeArray = MimeTypeArray;
        this.Plugin = Plugin;
        this.PluginArray = PluginArray;
        this.FileList = FileList;
        this.DOMRect = DOMRect;
        this.DOMRectReadOnly = DOMRectReadOnly;
        this.RadioNodeList = RadioNodeList;
        this.ValidityState = ValidityState;
        this.Headers = Headers;
        this.XMLHttpRequestUpload = XMLHttpRequestUpload;
        this.XMLHttpRequestEventTarget = XMLHttpRequestEventTarget;
        this.ReadableStream = ReadableStream;
        this.WritableStream = Stream.Writable;
        this.TransformStream = Stream.Transform;
        this.AbortController = AbortController;
        this.AbortSignal = AbortSignal;
        this.FormData = FormData;
        this.Permissions = Permissions;
        this.PermissionStatus = PermissionStatus;
        this.Clipboard = Clipboard;
        this.ClipboardItem = ClipboardItem;
        // Events
        this.onload = null;
        this.onerror = null;
        this.self = this;
        this.top = this;
        this.parent = this;
        this.window = this;
        this.globalThis = this;
        this.performance = performance;
        this.screenLeft = 0;
        this.screenTop = 0;
        this.screenX = 0;
        this.screenY = 0;
        this.crypto = webcrypto;
        this.closed = false;
        this.name = '';
        this.Buffer = Buffer;
        // Public internal properties
        // Used for tracking capture event listeners to improve performance when they are not used.
        // See EventTarget class.
        this[_b] = {};
        this[_c] = [];
        this[_d] = new DocumentReadyStateManager(this);
        this[_e] = null;
        // Private properties
        _BrowserWindow_browserFrame.set(this, void 0);
        _BrowserWindow_innerWidth.set(this, null);
        _BrowserWindow_innerHeight.set(this, null);
        _BrowserWindow_outerWidth.set(this, null);
        _BrowserWindow_outerHeight.set(this, null);
        _BrowserWindow_devicePixelRatio.set(this, null);
        _BrowserWindow_zeroTimeouts.set(this, null);
        const asyncTaskManager = browserFrame[PropertySymbol.asyncTaskManager];
        __classPrivateFieldSet(this, _BrowserWindow_browserFrame, browserFrame, "f");
        this.customElements = new CustomElementRegistry(this);
        this[PropertySymbol.navigator] = new Navigator(this);
        this[PropertySymbol.history] = new History();
        this[PropertySymbol.screen] = new Screen();
        this[PropertySymbol.sessionStorage] = StorageFactory.createStorage();
        this[PropertySymbol.localStorage] = StorageFactory.createStorage();
        this[PropertySymbol.location] = new Location(__classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f"), options?.url ?? 'about:blank');
        this[PropertySymbol.asyncTaskManager] = asyncTaskManager;
        this.console = browserFrame.page.console;
        WindowBrowserSettingsReader.setSettings(this, __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page.context.browser.settings);
        const window = this;
        this[PropertySymbol.setupVMContext]();
        // Class overrides
        // For classes that need to be bound to the correct context.
        /* eslint-disable jsdoc/require-jsdoc */
        class Request extends RequestImplementation {
            constructor(input, init) {
                super({ window, asyncTaskManager }, input, init);
            }
        }
        class Response extends ResponseImplementation {
            constructor(body, init) {
                super({ window, browserFrame }, body, init);
            }
        }
        _f = PropertySymbol.window;
        Response[_f] = window;
        class XMLHttpRequest extends XMLHttpRequestImplementation {
            constructor() {
                super({ window, browserFrame });
            }
        }
        class FileReader extends FileReaderImplementation {
            constructor() {
                super(window);
            }
        }
        class DOMParser extends DOMParserImplementation {
            constructor() {
                super(window);
            }
        }
        class Range extends RangeImplementation {
            constructor() {
                super(window);
            }
        }
        class HTMLScriptElement extends HTMLScriptElementImplementation {
            constructor() {
                super(browserFrame);
            }
        }
        class HTMLLinkElement extends HTMLLinkElementImplementation {
            constructor() {
                super(browserFrame);
            }
        }
        class HTMLIFrameElement extends HTMLIFrameElementImplementation {
            constructor() {
                super(browserFrame);
            }
        }
        class HTMLFormElement extends HTMLFormElementImplementation {
            constructor() {
                super(browserFrame);
            }
        }
        class Document extends DocumentImplementation {
            constructor() {
                super({ window, browserFrame });
            }
        }
        class HTMLDocument extends HTMLDocumentImplementation {
            constructor() {
                super({ window, browserFrame });
            }
        }
        class XMLDocument extends XMLDocumentImplementation {
            constructor() {
                super({ window, browserFrame });
            }
        }
        class SVGDocument extends SVGDocumentImplementation {
            constructor() {
                super({ window, browserFrame });
            }
        }
        class Audio extends AudioImplementation {
        }
        class Image extends ImageImplementation {
        }
        class DocumentFragment extends DocumentFragmentImplementation {
        }
        class Text extends TextImplementation {
        }
        class Comment extends CommentImplementation {
        }
        /* eslint-enable jsdoc/require-jsdoc */
        this.Response = Response;
        this.Request = Request;
        this.Image = Image;
        this.Text = Text;
        this.Comment = Comment;
        this.DocumentFragment = DocumentFragment;
        this.FileReader = FileReader;
        this.DOMParser = DOMParser;
        this.XMLHttpRequest = XMLHttpRequest;
        this.Range = Range;
        this.Audio = Audio;
        this.HTMLScriptElement = HTMLScriptElement;
        this.HTMLLinkElement = HTMLLinkElement;
        this.HTMLIFrameElement = HTMLIFrameElement;
        this.HTMLFormElement = HTMLFormElement;
        this.Document = Document;
        this.HTMLDocument = HTMLDocument;
        this.XMLDocument = XMLDocument;
        this.SVGDocument = SVGDocument;
        // Override owner document
        this.Document[PropertySymbol.ownerDocument] = null;
        this.HTMLDocument[PropertySymbol.ownerDocument] = null;
        this.XMLDocument[PropertySymbol.ownerDocument] = null;
        this.SVGDocument[PropertySymbol.ownerDocument] = null;
        // Document
        this.document = new HTMLDocument();
        this.document[PropertySymbol.defaultView] = this;
        // Override owner document
        this.Audio[PropertySymbol.ownerDocument] = this.document;
        this.Image[PropertySymbol.ownerDocument] = this.document;
        this.DocumentFragment[PropertySymbol.ownerDocument] = this.document;
        this.Text[PropertySymbol.ownerDocument] = this.document;
        this.Comment[PropertySymbol.ownerDocument] = this.document;
        // Ready state manager
        this[PropertySymbol.readyStateManager].waitUntilComplete().then(() => {
            this.document[PropertySymbol.readyState] = DocumentReadyStateEnum.complete;
            this.document.dispatchEvent(new Event('readystatechange'));
            this.document.dispatchEvent(new Event('load', { bubbles: true }));
        });
        __classPrivateFieldGet(this, _BrowserWindow_instances, "m", _BrowserWindow_bindToThisScope).call(this);
    }
    /**
     * Returns location.
     */
    get location() {
        return this[PropertySymbol.location];
    }
    /**
     * Returns location.
     *
     * @param href Href.
     */
    set location(href) {
        this[PropertySymbol.location].href = href;
    }
    /**
     * Returns history.
     */
    get history() {
        return this[PropertySymbol.history];
    }
    /**
     * Returns navigator.
     */
    get navigator() {
        return this[PropertySymbol.navigator];
    }
    /**
     * Returns screen.
     */
    get screen() {
        return this[PropertySymbol.screen];
    }
    /**
     * Returns session storage.
     */
    get sessionStorage() {
        return this[PropertySymbol.sessionStorage];
    }
    /**
     * Returns local storage.
     */
    get localStorage() {
        return this[PropertySymbol.localStorage];
    }
    /**
     * Returns opener.
     *
     * @returns Opener.
     */
    get opener() {
        return __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.openerWindow];
    }
    /**
     * The number of pixels that the document is currently scrolled horizontally.
     *
     * @returns Scroll X.
     */
    get scrollX() {
        return this.document?.documentElement?.scrollLeft ?? 0;
    }
    /**
     * The read-only Window property pageXOffset is an alias for scrollX.
     *
     * @returns Scroll X.
     */
    get pageXOffset() {
        return this.scrollX;
    }
    /**
     * The number of pixels that the document is currently scrolled vertically.
     *
     * @returns Scroll Y.
     */
    get scrollY() {
        return this.document?.documentElement?.scrollTop ?? 0;
    }
    /**
     * The read-only Window property pageYOffset is an alias for scrollY.
     *
     * @returns Scroll Y.
     */
    get pageYOffset() {
        return this.scrollY;
    }
    /**
     * The CSS interface holds useful CSS-related methods.
     *
     * @returns CSS interface.
     */
    get CSS() {
        return new CSS();
    }
    /**
     * Returns inner width.
     *
     * @returns Inner width.
     */
    get innerWidth() {
        if (__classPrivateFieldGet(this, _BrowserWindow_innerWidth, "f") === null) {
            return __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page.viewport.width;
        }
        return __classPrivateFieldGet(this, _BrowserWindow_innerWidth, "f");
    }
    /**
     * Sets inner width.
     *
     * @param value Inner width.
     */
    set innerWidth(value) {
        __classPrivateFieldSet(this, _BrowserWindow_innerWidth, value, "f");
    }
    /**
     * Returns inner height.
     *
     * @returns Inner height.
     */
    get innerHeight() {
        // It seems like this value can be defined according to spec, but changing it has no effect on the actual viewport.
        if (__classPrivateFieldGet(this, _BrowserWindow_innerHeight, "f") === null) {
            return __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page.viewport.height;
        }
        return __classPrivateFieldGet(this, _BrowserWindow_innerHeight, "f");
    }
    /**
     * Sets inner height.
     *
     * @param value Inner height.
     */
    set innerHeight(value) {
        __classPrivateFieldSet(this, _BrowserWindow_innerHeight, value, "f");
    }
    /**
     * Returns outer width.
     *
     * @returns Outer width.
     */
    get outerWidth() {
        // It seems like this value can be defined according to spec, but changing it has no effect on the actual viewport.
        if (__classPrivateFieldGet(this, _BrowserWindow_outerWidth, "f") === null) {
            return __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page.viewport.width;
        }
        return __classPrivateFieldGet(this, _BrowserWindow_outerWidth, "f");
    }
    /**
     * Sets outer width.
     *
     * @param value Outer width.
     */
    set outerWidth(value) {
        __classPrivateFieldSet(this, _BrowserWindow_outerWidth, value, "f");
    }
    /**
     * Returns outer height.
     *
     * @returns Outer height.
     */
    get outerHeight() {
        if (__classPrivateFieldGet(this, _BrowserWindow_outerHeight, "f") === null) {
            return __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page.viewport.height;
        }
        return __classPrivateFieldGet(this, _BrowserWindow_outerHeight, "f");
    }
    /**
     * Sets outer height.
     *
     * @param value Outer height.
     */
    set outerHeight(value) {
        __classPrivateFieldSet(this, _BrowserWindow_outerHeight, value, "f");
    }
    /**
     * Returns device pixel ratio.
     *
     * @returns Device pixel ratio.
     */
    get devicePixelRatio() {
        // It seems like this value can be defined according to spec, but changing it has no effect on the actual viewport.
        if (__classPrivateFieldGet(this, _BrowserWindow_devicePixelRatio, "f") === null) {
            return __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page.viewport.devicePixelRatio;
        }
        return __classPrivateFieldGet(this, _BrowserWindow_devicePixelRatio, "f");
    }
    /**
     * Sets device pixel ratio.
     *
     * @param value Device pixel ratio.
     */
    set devicePixelRatio(value) {
        __classPrivateFieldSet(this, _BrowserWindow_devicePixelRatio, value, "f");
    }
    /**
     * Returns an object containing the values of all CSS properties of an element.
     *
     * @param element Element.
     * @returns CSS style declaration.
     */
    getComputedStyle(element) {
        element[PropertySymbol.computedStyle] =
            element[PropertySymbol.computedStyle] || new CSSStyleDeclaration(element, true);
        return element[PropertySymbol.computedStyle];
    }
    /**
     * Returns selection.
     *
     * @returns Selection.
     */
    getSelection() {
        return this.document.getSelection();
    }
    /**
     * Scrolls to a particular set of coordinates.
     *
     * @param x X position or options object.
     * @param y Y position.
     */
    scroll(x, y) {
        if (typeof x === 'object') {
            if (x.behavior === 'smooth') {
                this.setTimeout(() => {
                    if (x.top !== undefined) {
                        this.document.documentElement.scrollTop = x.top;
                    }
                    if (x.left !== undefined) {
                        this.document.documentElement.scrollLeft = x.left;
                    }
                });
            }
            else {
                if (x.top !== undefined) {
                    this.document.documentElement.scrollTop = x.top;
                }
                if (x.left !== undefined) {
                    this.document.documentElement.scrollLeft = x.left;
                }
            }
        }
        else if (x !== undefined && y !== undefined) {
            this.document.documentElement.scrollLeft = x;
            this.document.documentElement.scrollTop = y;
        }
    }
    /**
     * Scrolls to a particular set of coordinates.
     *
     * @param x X position or options object.
     * @param y Y position.
     */
    scrollTo(x, y) {
        this.scroll(x, y);
    }
    /**
     * Shifts focus away from the window.
     */
    blur() {
        // TODO: Implement.
    }
    /**
     * Gives focus to the window.
     */
    focus() {
        // TODO: Implement.
    }
    /**
     * Loads a specified resource into a new or existing browsing context (that is, a tab, a window, or an iframe) under a specified name.
     *
     * @param [url] URL.
     * @param [target] Target.
     * @param [features] Window features.
     * @returns Window.
     */
    open(url, target, features) {
        return WindowPageOpenUtility.openPage(__classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f"), {
            url,
            target,
            features
        });
    }
    /**
     * Closes the window.
     */
    close() {
        // When using a Window instance directly, the Window instance is the main frame and we will close the page and destroy the browser.
        // When using the Browser API we should only close the page when the Window instance is connected to the main frame (we should not close child frames such as iframes).
        if (__classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page?.mainFrame === __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")) {
            this[PropertySymbol.destroy]();
            __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page.close();
        }
    }
    /**
     * Returns a new MediaQueryList object that can then be used to determine if the document matches the media query string.
     *
     * @param mediaQueryString A string specifying the media query to parse into a MediaQueryList.
     * @returns A new MediaQueryList.
     */
    matchMedia(mediaQueryString) {
        return new MediaQueryList({ ownerWindow: this, media: mediaQueryString });
    }
    /**
     * Sets a timer which executes a function once the timer expires.
     *
     * @param callback Function to be executed.
     * @param [delay=0] Delay in ms.
     * @param args Arguments passed to the callback function.
     * @returns Timeout ID.
     */
    setTimeout(callback, delay = 0, ...args) {
        // We can group timeouts with a delay of 0 into one timeout to improve performance.
        // Grouping timeouts will also improve the performance of the async task manager.
        // It may also make the async task manager to stable as many timeouts may cause waitUntilComplete() to be resolved to early.
        if (!delay) {
            if (!__classPrivateFieldGet(this, _BrowserWindow_zeroTimeouts, "f")) {
                const settings = __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page?.context?.browser?.settings;
                const useTryCatch = !settings ||
                    !settings.disableErrorCapturing ||
                    settings.errorCapture === BrowserErrorCaptureEnum.tryAndCatch;
                const id = TIMER.setTimeout(() => {
                    const zeroTimeouts = __classPrivateFieldGet(this, _BrowserWindow_zeroTimeouts, "f");
                    __classPrivateFieldSet(this, _BrowserWindow_zeroTimeouts, null, "f");
                    for (const zeroTimeout of zeroTimeouts) {
                        if (useTryCatch) {
                            WindowErrorUtility.captureError(this, () => zeroTimeout.callback());
                        }
                        else {
                            zeroTimeout.callback();
                        }
                    }
                    __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTimer(id);
                });
                __classPrivateFieldSet(this, _BrowserWindow_zeroTimeouts, [], "f");
                __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].startTimer(id);
            }
            const zeroTimeout = new Timeout(() => callback(...args));
            __classPrivateFieldGet(this, _BrowserWindow_zeroTimeouts, "f").push(zeroTimeout);
            return zeroTimeout;
        }
        const settings = __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page?.context?.browser?.settings;
        const useTryCatch = !settings ||
            !settings.disableErrorCapturing ||
            settings.errorCapture === BrowserErrorCaptureEnum.tryAndCatch;
        const id = TIMER.setTimeout(() => {
            if (useTryCatch) {
                WindowErrorUtility.captureError(this, () => callback(...args));
            }
            else {
                callback(...args);
            }
            __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTimer(id);
        }, settings?.timer.maxTimeout !== -1 && delay && delay > settings?.timer.maxTimeout
            ? settings?.timer.maxTimeout
            : delay);
        __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].startTimer(id);
        return id;
    }
    /**
     * Cancels a timeout previously established by calling setTimeout().
     *
     * @param id ID of the timeout.
     */
    clearTimeout(id) {
        if (id && id instanceof Timeout) {
            const zeroTimeouts = __classPrivateFieldGet(this, _BrowserWindow_zeroTimeouts, "f") || [];
            const index = zeroTimeouts.indexOf(id);
            if (index !== -1) {
                zeroTimeouts.splice(index, 1);
            }
            return;
        }
        // We need to make sure that the ID is a Timeout object, otherwise Node.js might throw an error.
        // This is only necessary if we are in a Node.js environment.
        if (IS_NODE_JS_TIMEOUT_ENVIRONMENT && (!id || id.constructor.name !== 'Timeout')) {
            return;
        }
        TIMER.clearTimeout(id);
        __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTimer(id);
    }
    /**
     * Calls a function with a fixed time delay between each call.
     *
     * @param callback Function to be executed.
     * @param [delay=0] Delay in ms.
     * @param args Arguments passed to the callback function.
     * @returns Interval ID.
     */
    setInterval(callback, delay = 0, ...args) {
        const settings = __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page?.context?.browser?.settings;
        const useTryCatch = !settings ||
            !settings.disableErrorCapturing ||
            settings.errorCapture === BrowserErrorCaptureEnum.tryAndCatch;
        let iterations = 0;
        const id = TIMER.setInterval(() => {
            if (useTryCatch) {
                WindowErrorUtility.captureError(this, () => callback(...args), () => this.clearInterval(id));
            }
            else {
                callback(...args);
            }
            if (settings?.timer.maxIntervalIterations !== -1) {
                if (iterations >= settings?.timer.maxIntervalIterations) {
                    this.clearInterval(id);
                }
                iterations++;
            }
        }, settings?.timer.maxIntervalTime !== -1 && delay && delay > settings?.timer.maxIntervalTime
            ? settings?.timer.maxIntervalTime
            : delay);
        __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].startTimer(id);
        return id;
    }
    /**
     * Cancels a timed repeating action which was previously established by a call to setInterval().
     *
     * @param id ID of the interval.
     */
    clearInterval(id) {
        // We need to make sure that the ID is a Timeout object, otherwise Node.js might throw an error.
        // This is only necessary if we are in a Node.js environment.
        if (IS_NODE_JS_TIMEOUT_ENVIRONMENT && (!id || id.constructor.name !== 'Timeout')) {
            return;
        }
        TIMER.clearInterval(id);
        __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTimer(id);
    }
    /**
     * Mock animation frames with timeouts.
     *
     * @param callback Callback.
     * @returns ID.
     */
    requestAnimationFrame(callback) {
        const settings = __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page?.context?.browser?.settings;
        const useTryCatch = !settings ||
            !settings.disableErrorCapturing ||
            settings.errorCapture === BrowserErrorCaptureEnum.tryAndCatch;
        const id = TIMER.setImmediate(() => {
            if (useTryCatch) {
                WindowErrorUtility.captureError(this, () => callback(this.performance.now()));
            }
            else {
                callback(this.performance.now());
            }
            __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].endImmediate(id);
        });
        __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].startImmediate(id);
        return id;
    }
    /**
     * Mock animation frames with timeouts.
     *
     * @param id ID.
     */
    cancelAnimationFrame(id) {
        // We need to make sure that the ID is an Immediate object, otherwise Node.js might throw an error.
        // This is only necessary if we are in a Node.js environment.
        if (IS_NODE_JS_TIMEOUT_ENVIRONMENT && (!id || id.constructor.name !== 'Immediate')) {
            return;
        }
        TIMER.clearImmediate(id);
        __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].endImmediate(id);
    }
    /**
     * Queues a microtask to be executed at a safe time prior to control returning to the browser's event loop.
     *
     * @param callback Function to be executed.
     */
    queueMicrotask(callback) {
        let isAborted = false;
        const taskId = __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].startTask(() => (isAborted = true));
        const settings = __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page?.context?.browser?.settings;
        const useTryCatch = !settings ||
            !settings.disableErrorCapturing ||
            settings.errorCapture === BrowserErrorCaptureEnum.tryAndCatch;
        TIMER.queueMicrotask(() => {
            if (!isAborted) {
                if (useTryCatch) {
                    WindowErrorUtility.captureError(this, callback);
                }
                else {
                    callback();
                }
                __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskId);
            }
        });
    }
    /**
     * This method provides an easy, logical way to fetch resources asynchronously across the network.
     *
     * @param url URL.
     * @param [init] Init.
     * @returns Promise.
     */
    async fetch(url, init) {
        return await new Fetch({
            browserFrame: __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f"),
            window: this,
            url,
            init
        }).send();
    }
    /**
     * Creates a Base64-encoded ASCII string from a binary string (i.e., a string in which each character in the string is treated as a byte of binary data).
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/btoa
     * @param data Binay data.
     * @returns Base64-encoded string.
     */
    btoa(data) {
        return Base64.btoa(data);
    }
    /**
     * Decodes a string of data which has been encoded using Base64 encoding.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/atob
     * @see https://infra.spec.whatwg.org/#forgiving-base64-encode.
     * @see Https://html.spec.whatwg.org/multipage/webappapis.html#btoa.
     * @param data Binay string.
     * @returns An ASCII string containing decoded data from encodedData.
     */
    atob(data) {
        return Base64.atob(data);
    }
    /**
     * Safely enables cross-origin communication between Window objects; e.g., between a page and a pop-up that it spawned, or between a page and an iframe embedded within it.
     *
     * @param message Message.
     * @param [targetOrigin=*] Target origin.
     * @param _transfer Transfer. Not implemented.
     */
    postMessage(message, targetOrigin = '*', _transfer) {
        // TODO: Implement transfer.
        if (targetOrigin && targetOrigin !== '*' && this.location.origin !== targetOrigin) {
            throw new DOMException(`Failed to execute 'postMessage' on 'Window': The target origin provided ('${targetOrigin}') does not match the recipient window\'s origin ('${this.location.origin}').`, DOMExceptionNameEnum.securityError);
        }
        try {
            JSON.stringify(message);
        }
        catch (error) {
            throw new DOMException(`Failed to execute 'postMessage' on 'Window': The provided message cannot be serialized.`, DOMExceptionNameEnum.invalidStateError);
        }
        this.setTimeout(() => this.dispatchEvent(new MessageEvent('message', {
            data: message,
            origin: __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").parentFrame
                ? __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").parentFrame.window.location.origin
                : __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").window.location.origin,
            source: __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").parentFrame
                ? __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").parentFrame.window
                : __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").window,
            lastEventId: ''
        })));
    }
    /**
     * Resizes the window.
     *
     * @param width Width.
     * @param height Height.
     */
    resizeTo(width, height) {
        if (!width || !height) {
            throw new DOMException(`Failed to execute 'resizeTo' on 'Window': 2 arguments required, but only ${arguments.length} present.`);
        }
        // We can only resize the window if it is a popup.
        if (__classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.popup]) {
            __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page.setViewport({ width, height });
        }
    }
    /**
     * Resizes the current window by a specified amount.
     *
     * @param width Width.
     * @param height Height.
     */
    resizeBy(width, height) {
        if (!width || !height) {
            throw new DOMException(`Failed to execute 'resizeBy' on 'Window': 2 arguments required, but only ${arguments.length} present.`);
        }
        // We can only resize the window if it is a popup.
        if (__classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f")[PropertySymbol.popup]) {
            const viewport = __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page.viewport;
            __classPrivateFieldGet(this, _BrowserWindow_browserFrame, "f").page.setViewport({
                width: viewport.width + width,
                height: viewport.height + height
            });
        }
    }
    /**
     * Setup of VM context.
     */
    [(_BrowserWindow_browserFrame = new WeakMap(), _BrowserWindow_innerWidth = new WeakMap(), _BrowserWindow_innerHeight = new WeakMap(), _BrowserWindow_outerWidth = new WeakMap(), _BrowserWindow_outerHeight = new WeakMap(), _BrowserWindow_devicePixelRatio = new WeakMap(), _BrowserWindow_zeroTimeouts = new WeakMap(), _BrowserWindow_instances = new WeakSet(), _b = PropertySymbol.captureEventListenerCount, _c = PropertySymbol.mutationObservers, _d = PropertySymbol.readyStateManager, _e = PropertySymbol.asyncTaskManager, PropertySymbol.location, PropertySymbol.history, PropertySymbol.navigator, PropertySymbol.screen, PropertySymbol.sessionStorage, PropertySymbol.localStorage, PropertySymbol.setupVMContext)]() {
        if (!VM.isContext(this)) {
            VM.createContext(this);
            // Sets global properties from the VM to the Window object.
            // Otherwise "this.Array" will be undefined for example.
            VMGlobalPropertyScript.runInContext(this);
        }
    }
    /**
     * Destroys the window.
     */
    [PropertySymbol.destroy]() {
        if (!this.Audio[PropertySymbol.ownerDocument]) {
            return;
        }
        this.closed = true;
        this[PropertySymbol.asyncTaskManager] = null;
        this.Audio[PropertySymbol.ownerDocument] = null;
        this.Image[PropertySymbol.ownerDocument] = null;
        this.DocumentFragment[PropertySymbol.ownerDocument] = null;
        this.Text[PropertySymbol.ownerDocument] = null;
        this.Comment[PropertySymbol.ownerDocument] = null;
        const mutationObservers = this[PropertySymbol.mutationObservers];
        for (const mutationObserver of mutationObservers) {
            mutationObserver.disconnect();
        }
        this[PropertySymbol.mutationObservers] = [];
        // Disconnects nodes from the document, so that they can be garbage collected.
        for (const node of this.document[PropertySymbol.childNodes].slice()) {
            // Makes sure that something won't be triggered by the disconnect.
            if (node.disconnectedCallback) {
                delete node.disconnectedCallback;
            }
            this.document.removeChild(node);
        }
        if (this.customElements[PropertySymbol.destroy]) {
            this.customElements[PropertySymbol.destroy]();
        }
        this.document[PropertySymbol.activeElement] = null;
        this.document[PropertySymbol.nextActiveElement] = null;
        this.document[PropertySymbol.currentScript] = null;
        this.document[PropertySymbol.selection] = null;
        WindowBrowserSettingsReader.removeSettings(this);
    }
}
_a = BrowserWindow, _BrowserWindow_bindToThisScope = function _BrowserWindow_bindToThisScope() {
    const propertyDescriptors = Object.assign(Object.getOwnPropertyDescriptors(EventTarget.prototype), Object.getOwnPropertyDescriptors(_a.prototype));
    for (const key of Object.keys(propertyDescriptors)) {
        const descriptor = propertyDescriptors[key];
        if (descriptor.get || descriptor.set) {
            Object.defineProperty(this, key, {
                configurable: true,
                enumerable: true,
                get: descriptor.get?.bind(this),
                set: descriptor.set?.bind(this)
            });
        }
        else if (key !== 'constructor' &&
            key[0] !== '_' &&
            key[0] === key[0].toLowerCase() &&
            typeof this[key] === 'function' &&
            !this[key].toString().startsWith('class ')) {
            this[key] = this[key].bind(this);
        }
    }
};
export default BrowserWindow;
//# sourceMappingURL=BrowserWindow.js.map