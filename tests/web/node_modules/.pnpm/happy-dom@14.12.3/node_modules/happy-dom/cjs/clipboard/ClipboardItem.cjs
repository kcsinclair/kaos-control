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
var _ClipboardItem_data;
Object.defineProperty(exports, "__esModule", { value: true });
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
/**
 * Clipboard Item API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/ClipboardItem.
 */
class ClipboardItem {
    /**
     * Constructor.
     *
     * @param data Data.
     * @param [options] Options.
     * @param [options.presentationStyle] Presentation style.
     */
    constructor(data, options) {
        this.presentationStyle = 'unspecified';
        _ClipboardItem_data.set(this, void 0);
        __classPrivateFieldSet(this, _ClipboardItem_data, data, "f");
        if (options?.presentationStyle) {
            this.presentationStyle = options.presentationStyle;
        }
    }
    /**
     * Returns types.
     *
     * @returns Types.
     */
    get types() {
        return Object.keys(__classPrivateFieldGet(this, _ClipboardItem_data, "f"));
    }
    /**
     * Returns data by type.
     *
     * @param type Type.
     * @returns Data.
     */
    async getType(type) {
        if (!__classPrivateFieldGet(this, _ClipboardItem_data, "f")[type]) {
            throw new DOMException_js_1.default(`Failed to execute 'getType' on 'ClipboardItem': The type '${type}' was not found`);
        }
        return __classPrivateFieldGet(this, _ClipboardItem_data, "f")[type];
    }
}
_ClipboardItem_data = new WeakMap();
exports.default = ClipboardItem;
//# sourceMappingURL=ClipboardItem.cjs.map