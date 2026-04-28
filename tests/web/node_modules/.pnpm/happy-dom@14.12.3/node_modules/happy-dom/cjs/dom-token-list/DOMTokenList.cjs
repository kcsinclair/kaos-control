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
var _DOMTokenList_instances, _DOMTokenList_length, _DOMTokenList_ownerElement, _DOMTokenList_attributeName, _DOMTokenList_getTokenList;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const ATTRIBUTE_SPLIT_REGEXP = /[\t\f\n\r ]+/;
/**
 * DOM Token List.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DOMTokenList.
 */
class DOMTokenList {
    /**
     * Constructor.
     *
     * @param ownerElement Owner element.
     * @param attributeName Attribute name.
     */
    constructor(ownerElement, attributeName) {
        _DOMTokenList_instances.add(this);
        _DOMTokenList_length.set(this, 0);
        _DOMTokenList_ownerElement.set(this, void 0);
        _DOMTokenList_attributeName.set(this, void 0);
        __classPrivateFieldSet(this, _DOMTokenList_ownerElement, ownerElement, "f");
        __classPrivateFieldSet(this, _DOMTokenList_attributeName, attributeName, "f");
        this[PropertySymbol.updateIndices]();
    }
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length() {
        return __classPrivateFieldGet(this, _DOMTokenList_length, "f");
    }
    /**
     * Set value.
     *
     * @param value Value.
     */
    set value(value) {
        __classPrivateFieldGet(this, _DOMTokenList_ownerElement, "f").setAttribute(__classPrivateFieldGet(this, _DOMTokenList_attributeName, "f"), value);
    }
    /**
     * Get value.
     */
    get value() {
        return __classPrivateFieldGet(this, _DOMTokenList_ownerElement, "f").getAttribute(__classPrivateFieldGet(this, _DOMTokenList_attributeName, "f"));
    }
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     */
    [(_DOMTokenList_length = new WeakMap(), _DOMTokenList_ownerElement = new WeakMap(), _DOMTokenList_attributeName = new WeakMap(), _DOMTokenList_instances = new WeakSet(), Symbol.iterator)]() {
        return __classPrivateFieldGet(this, _DOMTokenList_instances, "m", _DOMTokenList_getTokenList).call(this).values();
    }
    /**
     * Get ClassName.
     *
     * @param index Index.
     * */
    item(index) {
        index = typeof index === 'number' ? index : 0;
        return index >= 0 && this[index] ? this[index] : null;
    }
    /**
     * Replace Token.
     *
     * @param token Token.
     * @param newToken NewToken.
     */
    replace(token, newToken) {
        const list = __classPrivateFieldGet(this, _DOMTokenList_instances, "m", _DOMTokenList_getTokenList).call(this);
        const index = list.indexOf(token);
        if (index === -1) {
            return false;
        }
        list[index] = newToken;
        __classPrivateFieldGet(this, _DOMTokenList_ownerElement, "f").setAttribute(__classPrivateFieldGet(this, _DOMTokenList_attributeName, "f"), list.join(' '));
        return true;
    }
    /**
     * Supports.
     *
     * @param _token Token.
     */
    supports(_token) {
        return false;
    }
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     */
    values() {
        return __classPrivateFieldGet(this, _DOMTokenList_instances, "m", _DOMTokenList_getTokenList).call(this).values();
    }
    /**
     * Returns an iterator, allowing you to go through all key/value pairs contained in this object.
     */
    entries() {
        return __classPrivateFieldGet(this, _DOMTokenList_instances, "m", _DOMTokenList_getTokenList).call(this).entries();
    }
    /**
     * Executes a provided callback function once for each DOMTokenList element.
     *
     * @param callback
     * @param thisArg
     */
    forEach(callback, thisArg) {
        return __classPrivateFieldGet(this, _DOMTokenList_instances, "m", _DOMTokenList_getTokenList).call(this).forEach(callback, thisArg);
    }
    /**
     * Returns an iterator, allowing you to go through all keys of the key/value pairs contained in this object.
     *
     */
    keys() {
        return __classPrivateFieldGet(this, _DOMTokenList_instances, "m", _DOMTokenList_getTokenList).call(this).keys();
    }
    /**
     * Adds tokens.
     *
     * @param tokens Tokens.
     */
    add(...tokens) {
        const list = __classPrivateFieldGet(this, _DOMTokenList_instances, "m", _DOMTokenList_getTokenList).call(this);
        for (const token of tokens) {
            const index = list.indexOf(token);
            if (index === -1) {
                list.push(token);
            }
            else {
                list[index] = token;
            }
        }
        __classPrivateFieldGet(this, _DOMTokenList_ownerElement, "f").setAttribute(__classPrivateFieldGet(this, _DOMTokenList_attributeName, "f"), list.join(' '));
    }
    /**
     * Removes tokens.
     *
     * @param tokens Tokens.
     */
    remove(...tokens) {
        const list = __classPrivateFieldGet(this, _DOMTokenList_instances, "m", _DOMTokenList_getTokenList).call(this);
        for (const token of tokens) {
            const index = list.indexOf(token);
            if (index !== -1) {
                list.splice(index, 1);
            }
        }
        __classPrivateFieldGet(this, _DOMTokenList_ownerElement, "f").setAttribute(__classPrivateFieldGet(this, _DOMTokenList_attributeName, "f"), list.join(' '));
    }
    /**
     * Check if the list contains a class.
     *
     * @param className Class name.
     * @returns TRUE if it contains.
     */
    contains(className) {
        const list = __classPrivateFieldGet(this, _DOMTokenList_instances, "m", _DOMTokenList_getTokenList).call(this);
        return list.includes(className);
    }
    /**
     * Toggle a class name.
     *
     * @param token A string representing the class name you want to toggle.
     * @param [force] If included, turns the toggle into a one way-only operation. If set to `false`, then class name will only be removed, but not added. If set to `true`, then class name will only be added, but not removed.
     * @returns A boolean value, `true` or `false`, indicating whether class name is in the list after the call or not.
     */
    toggle(token, force) {
        let shouldAdd;
        if (force !== undefined) {
            shouldAdd = force;
        }
        else {
            shouldAdd = !this.contains(token);
        }
        if (shouldAdd) {
            this.add(token);
            return true;
        }
        this.remove(token);
        return false;
    }
    /**
     * Updates indices.
     */
    [PropertySymbol.updateIndices]() {
        const list = __classPrivateFieldGet(this, _DOMTokenList_instances, "m", _DOMTokenList_getTokenList).call(this);
        for (let i = list.length - 1, max = this.length; i < max; i++) {
            delete this[i];
        }
        for (let i = 0, max = list.length; i < max; i++) {
            this[i] = list[i];
        }
        __classPrivateFieldSet(this, _DOMTokenList_length, list.length, "f");
    }
    /**
     * Returns DOMTokenList value.
     */
    toString() {
        return this.value || '';
    }
}
_DOMTokenList_getTokenList = function _DOMTokenList_getTokenList() {
    const attr = __classPrivateFieldGet(this, _DOMTokenList_ownerElement, "f").getAttribute(__classPrivateFieldGet(this, _DOMTokenList_attributeName, "f"));
    if (!attr) {
        return [];
    }
    // It is possible to make this statement shorter by using Array.from() and Set, but this is faster when comparing using a bench test.
    const list = [];
    for (const item of attr.trim().split(ATTRIBUTE_SPLIT_REGEXP)) {
        if (!list.includes(item)) {
            list.push(item);
        }
    }
    return list;
};
exports.default = DOMTokenList;
//# sourceMappingURL=DOMTokenList.cjs.map