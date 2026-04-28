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
var _CSSStyleSheet_currentText;
import DOMException from '../exception/DOMException.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import CSSParser from './utilities/CSSParser.js';
/**
 * CSS StyleSheet.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet.
 */
class CSSStyleSheet {
    /**
     * Constructor.
     *
     * @param [options] Options.
     * @param [options.media] Media.
     * @param [options.title] Title.
     * @param [options.alternate] Alternate.
     * @param [options.disabled] Disabled.
     */
    constructor(options) {
        this.value = null;
        this.name = null;
        this.namespaceURI = null;
        this.cssRules = [];
        _CSSStyleSheet_currentText.set(this, null);
        this.media = options && options.media ? options.media : '';
        this.title = options && options.title ? options.title : '';
        this.alternate = options && options.alternate ? options.alternate : false;
        this.disabled = options && options.disabled ? options.disabled : false;
    }
    /**
     * Inserts a rule.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/insertRule
     * @param rule Rule.
     * @param [index] Index.
     * @returns The newly inserterted rule's index.
     */
    insertRule(rule, index) {
        const rules = CSSParser.parseFromString(this, rule);
        if (rules.length === 0) {
            throw new DOMException('Invalid CSS rule.', DOMExceptionNameEnum.hierarchyRequestError);
        }
        if (rules.length > 1) {
            throw new DOMException('Only one rule is allowed to be added.', DOMExceptionNameEnum.syntaxError);
        }
        if (index !== undefined) {
            if (index > this.cssRules.length) {
                throw new DOMException('Index is more than the length of CSSRuleList.', DOMExceptionNameEnum.indexSizeError);
            }
            this.cssRules.splice(index, 0, rules[0]);
            return index;
        }
        const newIndex = this.cssRules.length;
        this.cssRules.push(rules[0]);
        return newIndex;
    }
    /**
     * Removes a rule.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/deleteRule
     * @param index Index.
     */
    deleteRule(index) {
        delete this.cssRules[index];
    }
    /**
     * Replaces all CSS rules.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/replace
     * @param text CSS text.
     * @returns Promise.
     */
    async replace(text) {
        this.replaceSync(text);
    }
    /**
     * Replaces all CSS rules.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/replaceSync
     * @param text CSS text.
     */
    replaceSync(text) {
        if (__classPrivateFieldGet(this, _CSSStyleSheet_currentText, "f") !== text) {
            __classPrivateFieldSet(this, _CSSStyleSheet_currentText, text, "f");
            this.cssRules = CSSParser.parseFromString(this, text);
        }
    }
}
_CSSStyleSheet_currentText = new WeakMap();
export default CSSStyleSheet;
//# sourceMappingURL=CSSStyleSheet.js.map