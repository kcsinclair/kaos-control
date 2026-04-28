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
var _AbstractCSSStyleDeclaration_style, _AbstractCSSStyleDeclaration_ownerElement, _AbstractCSSStyleDeclaration_computed, _AbstractCSSStyleDeclaration_elementStyle;
import * as PropertySymbol from '../../PropertySymbol.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import DOMException from '../../exception/DOMException.js';
import CSSStyleDeclarationElementStyle from './element-style/CSSStyleDeclarationElementStyle.js';
import CSSStyleDeclarationPropertyManager from './property-manager/CSSStyleDeclarationPropertyManager.js';
/**
 * CSS Style Declaration.
 */
class AbstractCSSStyleDeclaration {
    /**
     * Constructor.
     *
     * @param [ownerElement] Computed style element.
     * @param [computed] Computed.
     */
    constructor(ownerElement = null, computed = false) {
        this.parentRule = null;
        _AbstractCSSStyleDeclaration_style.set(this, null);
        _AbstractCSSStyleDeclaration_ownerElement.set(this, void 0);
        _AbstractCSSStyleDeclaration_computed.set(this, void 0);
        _AbstractCSSStyleDeclaration_elementStyle.set(this, null);
        __classPrivateFieldSet(this, _AbstractCSSStyleDeclaration_style, !ownerElement ? new CSSStyleDeclarationPropertyManager() : null, "f");
        __classPrivateFieldSet(this, _AbstractCSSStyleDeclaration_ownerElement, ownerElement, "f");
        __classPrivateFieldSet(this, _AbstractCSSStyleDeclaration_computed, ownerElement ? computed : false, "f");
        __classPrivateFieldSet(this, _AbstractCSSStyleDeclaration_elementStyle, ownerElement
            ? new CSSStyleDeclarationElementStyle(ownerElement, __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_computed, "f"))
            : null, "f");
    }
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length() {
        if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")) {
            const style = __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_elementStyle, "f").getElementStyle();
            return style.size();
        }
        return __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_style, "f").size();
    }
    /**
     * Returns the style decleration as a CSS text.
     *
     * @returns CSS text.
     */
    get cssText() {
        if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")) {
            if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_computed, "f")) {
                return '';
            }
            return __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_elementStyle, "f").getElementStyle().toString();
        }
        return __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_style, "f").toString();
    }
    /**
     * Sets CSS text.
     *
     * @param cssText CSS text.
     */
    set cssText(cssText) {
        if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_computed, "f")) {
            throw new DOMException(`Failed to execute 'cssText' on 'CSSStyleDeclaration': These styles are computed, and the properties are therefore read-only.`, DOMExceptionNameEnum.domException);
        }
        if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")) {
            const style = new CSSStyleDeclarationPropertyManager({ cssText });
            let styleAttribute = __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.attributes]['style'];
            if (!styleAttribute) {
                styleAttribute = __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.ownerDocument].createAttribute('style');
                // We use "[PropertySymbol.setNamedItemWithoutConsequences]" here to avoid triggering setting "Element.style.cssText" when setting the "style" attribute.
                __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.attributes][PropertySymbol.setNamedItemWithoutConsequences](styleAttribute);
            }
            if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.isConnected]) {
                __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.ownerDocument][PropertySymbol.cacheID]++;
            }
            styleAttribute[PropertySymbol.value] = style.toString();
        }
        else {
            __classPrivateFieldSet(this, _AbstractCSSStyleDeclaration_style, new CSSStyleDeclarationPropertyManager({ cssText }), "f");
        }
    }
    /**
     * Returns item.
     *
     * @param index Index.
     * @returns Item.
     */
    item(index) {
        if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")) {
            return __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_elementStyle, "f").getElementStyle().item(index);
        }
        return __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_style, "f").item(index);
    }
    /**
     * Set a property.
     *
     * @param name Property name.
     * @param value Value. Must not contain "!important" as that should be set using the priority parameter.
     * @param [priority] Can be "important", or an empty string.
     */
    setProperty(name, value, priority) {
        if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_computed, "f")) {
            throw new DOMException(`Failed to execute 'setProperty' on 'CSSStyleDeclaration': These styles are computed, and therefore the '${name}' property is read-only.`, DOMExceptionNameEnum.domException);
        }
        if (priority !== '' && priority !== undefined && priority !== 'important') {
            return;
        }
        const stringValue = String(value);
        if (!stringValue) {
            this.removeProperty(name);
        }
        else if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")) {
            let styleAttribute = __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.attributes]['style'];
            if (!styleAttribute) {
                styleAttribute = __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.ownerDocument].createAttribute('style');
                // We use "[PropertySymbol.setNamedItemWithoutConsequences]" here to avoid triggering setting "Element.style.cssText" when setting the "style" attribute.
                __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.attributes][PropertySymbol.setNamedItemWithoutConsequences](styleAttribute);
            }
            if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.isConnected]) {
                __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.ownerDocument][PropertySymbol.cacheID]++;
            }
            const style = __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_elementStyle, "f").getElementStyle();
            style.set(name, stringValue, !!priority);
            styleAttribute[PropertySymbol.value] = style.toString();
        }
        else {
            __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_style, "f").set(name, stringValue, !!priority);
        }
    }
    /**
     * Removes a property.
     *
     * @param name Property name in kebab case.
     * @param value Value. Must not contain "!important" as that should be set using the priority parameter.
     * @param [priority] Can be "important", or an empty string.
     */
    removeProperty(name) {
        if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_computed, "f")) {
            throw new DOMException(`Failed to execute 'removeProperty' on 'CSSStyleDeclaration': These styles are computed, and therefore the '${name}' property is read-only.`, DOMExceptionNameEnum.domException);
        }
        if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")) {
            const style = __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_elementStyle, "f").getElementStyle();
            style.remove(name);
            const newCSSText = style.toString();
            if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.isConnected]) {
                __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.ownerDocument][PropertySymbol.cacheID]++;
            }
            if (newCSSText) {
                __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.attributes]['style'][PropertySymbol.value] =
                    newCSSText;
            }
            else {
                // We use "[PropertySymbol.removeNamedItemWithoutConsequences]" here to avoid triggering setting "Element.style.cssText" when setting the "style" attribute.
                __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")[PropertySymbol.attributes][PropertySymbol.removeNamedItemWithoutConsequences]('style');
            }
        }
        else {
            __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_style, "f").remove(name);
        }
    }
    /**
     * Returns a property.
     *
     * @param name Property name in kebab case.
     * @returns Property value.
     */
    getPropertyValue(name) {
        if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")) {
            const style = __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_elementStyle, "f").getElementStyle();
            return style.get(name)?.value || '';
        }
        return __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_style, "f").get(name)?.value || '';
    }
    /**
     * Returns a property.
     *
     * @param name Property name in kebab case.
     * @returns "important" if set to be important.
     */
    getPropertyPriority(name) {
        if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")) {
            const style = __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_elementStyle, "f").getElementStyle();
            return style.get(name)?.important ? 'important' : '';
        }
        return __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_style, "f").get(name)?.important ? 'important' : '';
    }
}
_AbstractCSSStyleDeclaration_style = new WeakMap(), _AbstractCSSStyleDeclaration_ownerElement = new WeakMap(), _AbstractCSSStyleDeclaration_computed = new WeakMap(), _AbstractCSSStyleDeclaration_elementStyle = new WeakMap();
export default AbstractCSSStyleDeclaration;
//# sourceMappingURL=AbstractCSSStyleDeclaration.js.map