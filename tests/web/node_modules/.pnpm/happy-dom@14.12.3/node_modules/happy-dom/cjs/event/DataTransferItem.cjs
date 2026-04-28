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
var _DataTransferItem_item;
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Data transfer item.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DataTransferItem.
 */
class DataTransferItem {
    /**
     * Constructor.
     *
     * @param item Item.
     * @param type Type.
     */
    constructor(item, type = '') {
        _DataTransferItem_item.set(this, null);
        this.kind = typeof item === 'string' ? 'string' : 'file';
        this.type = this.kind === 'string' ? type : item.type;
        __classPrivateFieldSet(this, _DataTransferItem_item, item, "f");
    }
    /**
     * Returns file.
     */
    getAsFile() {
        if (this.kind === 'string') {
            return null;
        }
        return __classPrivateFieldGet(this, _DataTransferItem_item, "f");
    }
    /**
     * Returns string.
     *
     * @param callback Callback.
     */
    getAsString(callback) {
        if (this.kind === 'file') {
            callback;
        }
        callback(__classPrivateFieldGet(this, _DataTransferItem_item, "f"));
    }
}
_DataTransferItem_item = new WeakMap();
exports.default = DataTransferItem;
//# sourceMappingURL=DataTransferItem.cjs.map