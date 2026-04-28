"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _FileReader_instances, _FileReader_isTerminated, _FileReader_loadTimeout, _FileReader_parseTimeout, _FileReader_window, _FileReader_readFile;
Object.defineProperty(exports, "__esModule", { value: true });
const whatwg_mimetype_1 = __importDefault(require("whatwg-mimetype"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const ProgressEvent_js_1 = __importDefault(require("../event/events/ProgressEvent.cjs"));
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const FileReaderReadyStateEnum_js_1 = __importDefault(require("./FileReaderReadyStateEnum.cjs"));
const FileReaderFormatEnum_js_1 = __importDefault(require("./FileReaderFormatEnum.cjs"));
const EventTarget_js_1 = __importDefault(require("../event/EventTarget.cjs"));
const FileReaderEventTypeEnum_js_1 = __importDefault(require("./FileReaderEventTypeEnum.cjs"));
const buffer_1 = require("buffer");
/**
 * Reference:
 * https://developer.mozilla.org/sv-SE/docs/Web/API/FileReader.
 *
 * Based on:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/file-api/FileReader-impl.js (MIT licensed).
 */
class FileReader extends EventTarget_js_1.default {
    /**
     * Constructor.
     *
     * @param window Window.
     */
    constructor(window) {
        super();
        _FileReader_instances.add(this);
        this.error = null;
        this.result = null;
        this.readyState = FileReaderReadyStateEnum_js_1.default.empty;
        this.onabort = null;
        this.onerror = null;
        this.onload = null;
        this.onloadstart = null;
        this.onloadend = null;
        this.onprogress = null;
        _FileReader_isTerminated.set(this, false);
        _FileReader_loadTimeout.set(this, null);
        _FileReader_parseTimeout.set(this, null);
        _FileReader_window.set(this, void 0);
        __classPrivateFieldSet(this, _FileReader_window, window, "f");
    }
    /**
     * Reads as ArrayBuffer.
     *
     * @param blob Blob.
     */
    readAsArrayBuffer(blob) {
        __classPrivateFieldGet(this, _FileReader_instances, "m", _FileReader_readFile).call(this, blob, FileReaderFormatEnum_js_1.default.buffer);
    }
    /**
     * Reads as binary string.
     *
     * @param blob Blob.
     */
    readAsBinaryString(blob) {
        __classPrivateFieldGet(this, _FileReader_instances, "m", _FileReader_readFile).call(this, blob, FileReaderFormatEnum_js_1.default.binaryString);
    }
    /**
     * Reads as data URL.
     *
     * @param blob Blob.
     */
    readAsDataURL(blob) {
        __classPrivateFieldGet(this, _FileReader_instances, "m", _FileReader_readFile).call(this, blob, FileReaderFormatEnum_js_1.default.dataURL);
    }
    /**
     * Reads as text.
     *
     * @param blob Blob.
     * @param [encoding] Encoding.
     */
    readAsText(blob, encoding = null) {
        __classPrivateFieldGet(this, _FileReader_instances, "m", _FileReader_readFile).call(this, blob, FileReaderFormatEnum_js_1.default.text, encoding || 'UTF-8');
    }
    /**
     * Aborts the file reader.
     */
    abort() {
        __classPrivateFieldGet(this, _FileReader_window, "f").clearTimeout(__classPrivateFieldGet(this, _FileReader_loadTimeout, "f"));
        __classPrivateFieldGet(this, _FileReader_window, "f").clearTimeout(__classPrivateFieldGet(this, _FileReader_parseTimeout, "f"));
        if (this.readyState === FileReaderReadyStateEnum_js_1.default.empty ||
            this.readyState === FileReaderReadyStateEnum_js_1.default.done) {
            this.result = null;
            return;
        }
        if (this.readyState === FileReaderReadyStateEnum_js_1.default.loading) {
            this.readyState = FileReaderReadyStateEnum_js_1.default.done;
            this.result = null;
        }
        __classPrivateFieldSet(this, _FileReader_isTerminated, true, "f");
        this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.abort));
        this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.loadend));
    }
}
_FileReader_isTerminated = new WeakMap(), _FileReader_loadTimeout = new WeakMap(), _FileReader_parseTimeout = new WeakMap(), _FileReader_window = new WeakMap(), _FileReader_instances = new WeakSet(), _FileReader_readFile = function _FileReader_readFile(blob, format, encoding = null) {
    if (this.readyState === FileReaderReadyStateEnum_js_1.default.loading) {
        throw new DOMException_js_1.default('The object is in an invalid state.', DOMExceptionNameEnum_js_1.default.invalidStateError);
    }
    this.readyState = FileReaderReadyStateEnum_js_1.default.loading;
    __classPrivateFieldSet(this, _FileReader_loadTimeout, __classPrivateFieldGet(this, _FileReader_window, "f").setTimeout(() => {
        if (__classPrivateFieldGet(this, _FileReader_isTerminated, "f")) {
            __classPrivateFieldSet(this, _FileReader_isTerminated, false, "f");
            return;
        }
        this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.loadstart));
        let data = blob[PropertySymbol.buffer];
        if (!data) {
            data = buffer_1.Buffer.alloc(0);
        }
        this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.loadstart, {
            lengthComputable: !isNaN(blob.size),
            total: blob.size,
            loaded: data.length
        }));
        __classPrivateFieldSet(this, _FileReader_parseTimeout, __classPrivateFieldGet(this, _FileReader_window, "f").setTimeout(() => {
            if (__classPrivateFieldGet(this, _FileReader_isTerminated, "f")) {
                __classPrivateFieldSet(this, _FileReader_isTerminated, false, "f");
                return;
            }
            switch (format) {
                default:
                case FileReaderFormatEnum_js_1.default.buffer: {
                    this.result = new Uint8Array(data).buffer;
                    break;
                }
                case FileReaderFormatEnum_js_1.default.binaryString: {
                    this.result = data.toString('binary');
                    break;
                }
                case FileReaderFormatEnum_js_1.default.dataURL: {
                    // Spec seems very unclear here; see https://github.com/w3c/FileAPI/issues/104.
                    const contentType = whatwg_mimetype_1.default.parse(blob.type) || 'application/octet-stream';
                    this.result =
                        `data:${contentType};base64,${data.toString('base64')}`;
                    break;
                }
                case FileReaderFormatEnum_js_1.default.text: {
                    this.result = new TextDecoder(encoding || 'UTF-8').decode(data);
                    break;
                }
            }
            this.readyState = FileReaderReadyStateEnum_js_1.default.done;
            this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.load));
            this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.loadend));
        }), "f");
    }), "f");
};
exports.default = FileReader;
//# sourceMappingURL=FileReader.cjs.map