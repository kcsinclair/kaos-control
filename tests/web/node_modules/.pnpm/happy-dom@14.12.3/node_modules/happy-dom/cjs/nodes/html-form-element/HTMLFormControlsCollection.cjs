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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _a;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const RadioNodeList_js_1 = __importDefault(require("./RadioNodeList.cjs"));
/**
 * HTMLFormControlsCollection.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLFormControlsCollection
 */
class HTMLFormControlsCollection extends Array {
    constructor() {
        super(...arguments);
        this[_a] = {};
    }
    /**
     * Returns item by index.
     *
     * @param index Index.
     */
    item(index) {
        return index >= 0 && this[index] ? this[index] : null;
    }
    /**
     * Returns named item.
     *
     * @param name Name.
     * @returns Node.
     */
    namedItem(name) {
        if (this[PropertySymbol.namedItems][name] && this[PropertySymbol.namedItems][name].length) {
            if (this[PropertySymbol.namedItems][name].length === 1) {
                return this[PropertySymbol.namedItems][name][0];
            }
            return this[PropertySymbol.namedItems][name];
        }
        return null;
    }
    /**
     * Appends named item.
     *
     * @param node Node.
     * @param name Name.
     */
    [(_a = PropertySymbol.namedItems, PropertySymbol.appendNamedItem)](node, name) {
        if (name) {
            this[PropertySymbol.namedItems][name] =
                this[PropertySymbol.namedItems][name] || new RadioNodeList_js_1.default();
            if (!this[PropertySymbol.namedItems][name].includes(node)) {
                this[PropertySymbol.namedItems][name].push(node);
            }
            if (this[PropertySymbol.isValidPropertyName](name)) {
                this[name] =
                    this[PropertySymbol.namedItems][name].length > 1
                        ? this[PropertySymbol.namedItems][name]
                        : this[PropertySymbol.namedItems][name][0];
            }
        }
    }
    /**
     * Appends named item.
     *
     * @param node Node.
     * @param name Name.
     */
    [PropertySymbol.removeNamedItem](node, name) {
        if (name && this[PropertySymbol.namedItems][name]) {
            const index = this[PropertySymbol.namedItems][name].indexOf(node);
            if (index > -1) {
                this[PropertySymbol.namedItems][name].splice(index, 1);
                if (this[PropertySymbol.namedItems][name].length === 0) {
                    delete this[PropertySymbol.namedItems][name];
                    if (this.hasOwnProperty(name) && this[PropertySymbol.isValidPropertyName](name)) {
                        delete this[name];
                    }
                }
                else if (this[PropertySymbol.isValidPropertyName](name)) {
                    this[name] =
                        this[PropertySymbol.namedItems][name].length > 1
                            ? this[PropertySymbol.namedItems][name]
                            : this[PropertySymbol.namedItems][name][0];
                }
            }
        }
    }
    /**
     * Returns "true" if the property name is valid.
     *
     * @param name Name.
     * @returns True if the property name is valid.
     */
    [PropertySymbol.isValidPropertyName](name) {
        return (!!name &&
            !this.constructor.prototype.hasOwnProperty(name) &&
            !Array.prototype.hasOwnProperty(name) &&
            (isNaN(Number(name)) || name.includes('.')));
    }
}
exports.default = HTMLFormControlsCollection;
//# sourceMappingURL=HTMLFormControlsCollection.cjs.map