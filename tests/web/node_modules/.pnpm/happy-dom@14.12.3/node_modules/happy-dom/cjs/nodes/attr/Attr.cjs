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
var _a, _b, _c, _d, _e, _f;
Object.defineProperty(exports, "__esModule", { value: true });
const Node_js_1 = __importDefault(require("../node/Node.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../node/NodeTypeEnum.cjs"));
/**
 * Attribute node interface.
 *
 * Reference: https://developer.mozilla.org/en-US/docs/Web/API/Attr.
 */
class Attr extends Node_js_1.default {
    constructor() {
        super(...arguments);
        this[_a] = NodeTypeEnum_js_1.default.attributeNode;
        this[_b] = null;
        this[_c] = null;
        this[_d] = null;
        this[_e] = true;
        this[_f] = null;
    }
    /**
     * Returns specified.
     *
     * @returns Specified.
     */
    get specified() {
        return this[PropertySymbol.specified];
    }
    /**
     * Returns owner element.
     *
     * @returns Owner element.
     */
    get ownerElement() {
        return this[PropertySymbol.ownerElement];
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        return this[PropertySymbol.value];
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        this[PropertySymbol.value] = value;
    }
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this[PropertySymbol.name];
    }
    /**
     * Returns local name.
     *
     * @returns Local name.
     */
    get localName() {
        return this[PropertySymbol.name] ? this[PropertySymbol.name].split(':').reverse()[0] : null;
    }
    /**
     * Returns prefix.
     *
     * @returns Prefix.
     */
    get prefix() {
        return this[PropertySymbol.name] ? this[PropertySymbol.name].split(':')[0] : null;
    }
    /**
     * @override
     */
    get textContent() {
        return this[PropertySymbol.value];
    }
    /**
     * Returns namespace URI.
     *
     * @returns Namespace URI.
     */
    get namespaceURI() {
        return this[PropertySymbol.namespaceURI];
    }
}
_a = PropertySymbol.nodeType, _b = PropertySymbol.namespaceURI, _c = PropertySymbol.name, _d = PropertySymbol.value, _e = PropertySymbol.specified, _f = PropertySymbol.ownerElement;
exports.default = Attr;
//# sourceMappingURL=Attr.cjs.map