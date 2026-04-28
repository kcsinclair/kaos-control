"use strict";
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
var _Clipboard_ownerWindow, _Clipboard_data;
Object.defineProperty(exports, "__esModule", { value: true });
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const ClipboardItem_js_1 = __importDefault(require("./ClipboardItem.cjs"));
const Blob_js_1 = __importDefault(require("../file/Blob.cjs"));
/**
 * Clipboard API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Clipboard.
 */
class Clipboard {
    /**
     * Constructor.
     *
     * @param ownerWindow Owner window.
     */
    constructor(ownerWindow) {
        _Clipboard_ownerWindow.set(this, void 0);
        _Clipboard_data.set(this, []);
        __classPrivateFieldSet(this, _Clipboard_ownerWindow, ownerWindow, "f");
    }
    /**
     * Returns data.
     *
     * @returns Data.
     */
    async read() {
        const permissionStatus = await __classPrivateFieldGet(this, _Clipboard_ownerWindow, "f").navigator.permissions.query({
            name: 'clipboard-read'
        });
        if (permissionStatus.state === 'denied') {
            throw new DOMException_js_1.default(`Failed to execute 'read' on 'Clipboard': The request is not allowed`);
        }
        return __classPrivateFieldGet(this, _Clipboard_data, "f");
    }
    /**
     * Returns text.
     *
     * @returns Text.
     */
    async readText() {
        const permissionStatus = await __classPrivateFieldGet(this, _Clipboard_ownerWindow, "f").navigator.permissions.query({
            name: 'clipboard-read'
        });
        if (permissionStatus.state === 'denied') {
            throw new DOMException_js_1.default(`Failed to execute 'readText' on 'Clipboard': The request is not allowed`);
        }
        let text = '';
        for (const item of __classPrivateFieldGet(this, _Clipboard_data, "f")) {
            if (item.types.includes('text/plain')) {
                const data = await item.getType('text/plain');
                if (typeof data === 'string') {
                    text += data;
                }
                else {
                    // Instance of Blob
                    text += await data.text();
                }
            }
        }
        return text;
    }
    /**
     * Writes data.
     *
     * @param data Data.
     */
    async write(data) {
        const permissionStatus = await __classPrivateFieldGet(this, _Clipboard_ownerWindow, "f").navigator.permissions.query({
            name: 'clipboard-write'
        });
        if (permissionStatus.state === 'denied') {
            throw new DOMException_js_1.default(`Failed to execute 'write' on 'Clipboard': The request is not allowed`);
        }
        __classPrivateFieldSet(this, _Clipboard_data, data, "f");
    }
    /**
     * Writes text.
     *
     * @param text Text.
     */
    async writeText(text) {
        const permissionStatus = await __classPrivateFieldGet(this, _Clipboard_ownerWindow, "f").navigator.permissions.query({
            name: 'clipboard-write'
        });
        if (permissionStatus.state === 'denied') {
            throw new DOMException_js_1.default(`Failed to execute 'writeText' on 'Clipboard': The request is not allowed`);
        }
        __classPrivateFieldSet(this, _Clipboard_data, [new ClipboardItem_js_1.default({ 'text/plain': new Blob_js_1.default([text], { type: 'text/plain' }) })], "f");
    }
}
_Clipboard_ownerWindow = new WeakMap(), _Clipboard_data = new WeakMap();
exports.default = Clipboard;
//# sourceMappingURL=Clipboard.cjs.map