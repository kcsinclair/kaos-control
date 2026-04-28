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
var _a;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
/**
 * Storage.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Storage
 */
class Storage {
    constructor() {
        this[_a] = {};
    }
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length() {
        return Object.keys(this[PropertySymbol.data]).length;
    }
    /**
     * Returns name of the nth key.
     *
     * @param index Index.
     * @returns Name.
     */
    key(index) {
        const name = Object.keys(this[PropertySymbol.data])[index];
        return name !== undefined ? name : null;
    }
    /**
     * Sets item.
     *
     * @param name Name.
     * @param item Item.
     */
    setItem(name, item) {
        this[PropertySymbol.data][name] = String(item);
    }
    /**
     * Returns item.
     *
     * @param name Name.
     * @returns Item.
     */
    getItem(name) {
        return this[PropertySymbol.data][name] !== undefined ? this[PropertySymbol.data][name] : null;
    }
    /**
     * Removes item.
     *
     * @param name Name.
     */
    removeItem(name) {
        delete this[PropertySymbol.data][name];
    }
    /**
     * Clears storage.
     */
    clear() {
        this[PropertySymbol.data] = {};
    }
}
_a = PropertySymbol.data;
exports.default = Storage;
//# sourceMappingURL=Storage.cjs.map