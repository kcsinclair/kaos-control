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
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _CSSFontFaceRule_style, _a;
Object.defineProperty(exports, "__esModule", { value: true });
const CSSRule_js_1 = __importDefault(require("../CSSRule.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const CSSStyleDeclaration_js_1 = __importDefault(require("../declaration/CSSStyleDeclaration.cjs"));
/**
 * CSSRule interface.
 */
class CSSFontFaceRule extends CSSRule_js_1.default {
    constructor() {
        super(...arguments);
        this.type = CSSRule_js_1.default.FONT_FACE_RULE;
        this[_a] = '';
        _CSSFontFaceRule_style.set(this, null);
    }
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get style() {
        if (!__classPrivateFieldGet(this, _CSSFontFaceRule_style, "f")) {
            __classPrivateFieldSet(this, _CSSFontFaceRule_style, new CSSStyleDeclaration_js_1.default(), "f");
            __classPrivateFieldGet(this, _CSSFontFaceRule_style, "f").parentRule = this;
            __classPrivateFieldGet(this, _CSSFontFaceRule_style, "f").cssText = this[PropertySymbol.cssText];
        }
        return __classPrivateFieldGet(this, _CSSFontFaceRule_style, "f");
    }
}
_CSSFontFaceRule_style = new WeakMap(), _a = PropertySymbol.cssText;
exports.default = CSSFontFaceRule;
//# sourceMappingURL=CSSFontFaceRule.cjs.map