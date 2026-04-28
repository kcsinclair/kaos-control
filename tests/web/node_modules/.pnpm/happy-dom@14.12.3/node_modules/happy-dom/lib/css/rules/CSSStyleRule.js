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
var _CSSStyleRule_style, _a;
import CSSRule from '../CSSRule.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CSSStyleDeclaration from '../declaration/CSSStyleDeclaration.js';
/**
 * CSSRule interface.
 */
class CSSStyleRule extends CSSRule {
    constructor() {
        super(...arguments);
        this.type = CSSRule.STYLE_RULE;
        this.selectorText = '';
        this.styleMap = new Map();
        this[_a] = '';
        _CSSStyleRule_style.set(this, null);
    }
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get style() {
        if (!__classPrivateFieldGet(this, _CSSStyleRule_style, "f")) {
            __classPrivateFieldSet(this, _CSSStyleRule_style, new CSSStyleDeclaration(), "f");
            __classPrivateFieldGet(this, _CSSStyleRule_style, "f").parentRule = this;
            __classPrivateFieldGet(this, _CSSStyleRule_style, "f").cssText = this[PropertySymbol.cssText];
        }
        return __classPrivateFieldGet(this, _CSSStyleRule_style, "f");
    }
    /**
     * Returns css text.
     *
     * @returns CSS text.
     */
    get cssText() {
        return `${this.selectorText} { ${this.style.cssText} }`;
    }
}
_CSSStyleRule_style = new WeakMap(), _a = PropertySymbol.cssText;
export default CSSStyleRule;
//# sourceMappingURL=CSSStyleRule.js.map