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
var _CSSFontFaceRule_style, _a;
import CSSRule from '../CSSRule.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CSSStyleDeclaration from '../declaration/CSSStyleDeclaration.js';
/**
 * CSSRule interface.
 */
class CSSFontFaceRule extends CSSRule {
    constructor() {
        super(...arguments);
        this.type = CSSRule.FONT_FACE_RULE;
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
            __classPrivateFieldSet(this, _CSSFontFaceRule_style, new CSSStyleDeclaration(), "f");
            __classPrivateFieldGet(this, _CSSFontFaceRule_style, "f").parentRule = this;
            __classPrivateFieldGet(this, _CSSFontFaceRule_style, "f").cssText = this[PropertySymbol.cssText];
        }
        return __classPrivateFieldGet(this, _CSSFontFaceRule_style, "f");
    }
}
_CSSFontFaceRule_style = new WeakMap(), _a = PropertySymbol.cssText;
export default CSSFontFaceRule;
//# sourceMappingURL=CSSFontFaceRule.js.map