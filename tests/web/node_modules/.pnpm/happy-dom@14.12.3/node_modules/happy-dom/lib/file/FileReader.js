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
var _FileReader_instances, _FileReader_isTerminated, _FileReader_loadTimeout, _FileReader_parseTimeout, _FileReader_window, _FileReader_readFile;
import WhatwgMIMEType from 'whatwg-mimetype';
import * as PropertySymbol from '../PropertySymbol.js';
import ProgressEvent from '../event/events/ProgressEvent.js';
import DOMException from '../exception/DOMException.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import FileReaderReadyStateEnum from './FileReaderReadyStateEnum.js';
import FileReaderFormatEnum from './FileReaderFormatEnum.js';
import EventTarget from '../event/EventTarget.js';
import FileReaderEventTypeEnum from './FileReaderEventTypeEnum.js';
import { Buffer } from 'buffer';
/**
 * Reference:
 * https://developer.mozilla.org/sv-SE/docs/Web/API/FileReader.
 *
 * Based on:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/file-api/FileReader-impl.js (MIT licensed).
 */
class FileReader extends EventTarget {
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
        this.readyState = FileReaderReadyStateEnum.empty;
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
        __classPrivateFieldGet(this, _FileReader_instances, "m", _FileReader_readFile).call(this, blob, FileReaderFormatEnum.buffer);
    }
    /**
     * Reads as binary string.
     *
     * @param blob Blob.
     */
    readAsBinaryString(blob) {
        __classPrivateFieldGet(this, _FileReader_instances, "m", _FileReader_readFile).call(this, blob, FileReaderFormatEnum.binaryString);
    }
    /**
     * Reads as data URL.
     *
     * @param blob Blob.
     */
    readAsDataURL(blob) {
        __classPrivateFieldGet(this, _FileReader_instances, "m", _FileReader_readFile).call(this, blob, FileReaderFormatEnum.dataURL);
    }
    /**
     * Reads as text.
     *
     * @param blob Blob.
     * @param [encoding] Encoding.
     */
    readAsText(blob, encoding = null) {
        __classPrivateFieldGet(this, _FileReader_instances, "m", _FileReader_readFile).call(this, blob, FileReaderFormatEnum.text, encoding || 'UTF-8');
    }
    /**
     * Aborts the file reader.
     */
    abort() {
        __classPrivateFieldGet(this, _FileReader_window, "f").clearTimeout(__classPrivateFieldGet(this, _FileReader_loadTimeout, "f"));
        __classPrivateFieldGet(this, _FileReader_window, "f").clearTimeout(__classPrivateFieldGet(this, _FileReader_parseTimeout, "f"));
        if (this.readyState === FileReaderReadyStateEnum.empty ||
            this.readyState === FileReaderReadyStateEnum.done) {
            this.result = null;
            return;
        }
        if (this.readyState === FileReaderReadyStateEnum.loading) {
            this.readyState = FileReaderReadyStateEnum.done;
            this.result = null;
        }
        __classPrivateFieldSet(this, _FileReader_isTerminated, true, "f");
        this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.abort));
        this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.loadend));
    }
}
_FileReader_isTerminated = new WeakMap(), _FileReader_loadTimeout = new WeakMap(), _FileReader_parseTimeout = new WeakMap(), _FileReader_window = new WeakMap(), _FileReader_instances = new WeakSet(), _FileReader_readFile = function _FileReader_readFile(blob, format, encoding = null) {
    if (this.readyState === FileReaderReadyStateEnum.loading) {
        throw new DOMException('The object is in an invalid state.', DOMExceptionNameEnum.invalidStateError);
    }
    this.readyState = FileReaderReadyStateEnum.loading;
    __classPrivateFieldSet(this, _FileReader_loadTimeout, __classPrivateFieldGet(this, _FileReader_window, "f").setTimeout(() => {
        if (__classPrivateFieldGet(this, _FileReader_isTerminated, "f")) {
            __classPrivateFieldSet(this, _FileReader_isTerminated, false, "f");
            return;
        }
        this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.loadstart));
        let data = blob[PropertySymbol.buffer];
        if (!data) {
            data = Buffer.alloc(0);
        }
        this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.loadstart, {
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
                case FileReaderFormatEnum.buffer: {
                    this.result = new Uint8Array(data).buffer;
                    break;
                }
                case FileReaderFormatEnum.binaryString: {
                    this.result = data.toString('binary');
                    break;
                }
                case FileReaderFormatEnum.dataURL: {
                    // Spec seems very unclear here; see https://github.com/w3c/FileAPI/issues/104.
                    const contentType = WhatwgMIMEType.parse(blob.type) || 'application/octet-stream';
                    this.result =
                        `data:${contentType};base64,${data.toString('base64')}`;
                    break;
                }
                case FileReaderFormatEnum.text: {
                    this.result = new TextDecoder(encoding || 'UTF-8').decode(data);
                    break;
                }
            }
            this.readyState = FileReaderReadyStateEnum.done;
            this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.load));
            this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.loadend));
        }), "f");
    }), "f");
};
export default FileReader;
//# sourceMappingURL=FileReader.js.map