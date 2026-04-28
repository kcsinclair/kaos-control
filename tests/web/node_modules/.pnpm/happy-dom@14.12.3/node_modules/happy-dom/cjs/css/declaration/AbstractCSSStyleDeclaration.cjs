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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _AbstractCSSStyleDeclaration_style, _AbstractCSSStyleDeclaration_ownerElement, _AbstractCSSStyleDeclaration_computed, _AbstractCSSStyleDeclaration_elementStyle;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../../exception/DOMExceptionNameEnum.cjs"));
const DOMException_js_1 = __importDefault(require("../../exception/DOMException.cjs"));
const CSSStyleDeclarationElementStyle_js_1 = __importDefault(require("./element-style/CSSStyleDeclarationElementStyle.cjs"));
const CSSStyleDeclarationPropertyManager_js_1 = __importDefault(require("./property-manager/CSSStyleDeclarationPropertyManager.cjs"));
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
        __classPrivateFieldSet(this, _AbstractCSSStyleDeclaration_style, !ownerElement ? new CSSStyleDeclarationPropertyManager_js_1.default() : null, "f");
        __classPrivateFieldSet(this, _AbstractCSSStyleDeclaration_ownerElement, ownerElement, "f");
        __classPrivateFieldSet(this, _AbstractCSSStyleDeclaration_computed, ownerElement ? computed : false, "f");
        __classPrivateFieldSet(this, _AbstractCSSStyleDeclaration_elementStyle, ownerElement
            ? new CSSStyleDeclarationElementStyle_js_1.default(ownerElement, __classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_computed, "f"))
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
            throw new DOMException_js_1.default(`Failed to execute 'cssText' on 'CSSStyleDeclaration': These styles are computed, and the properties are therefore read-only.`, DOMExceptionNameEnum_js_1.default.domException);
        }
        if (__classPrivateFieldGet(this, _AbstractCSSStyleDeclaration_ownerElement, "f")) {
            const style = new CSSStyleDeclarationPropertyManager_js_1.default({ cssText });
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
            __classPrivateFieldSet(this, _AbstractCSSStyleDeclaration_style, new CSSStyleDeclarationPropertyManager_js_1.default({ cssText }), "f");
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
            throw new DOMException_js_1.default(`Failed to execute 'setProperty' on 'CSSStyleDeclaration': These styles are computed, and therefore the '${name}' property is read-only.`, DOMExceptionNameEnum_js_1.default.domException);
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
            throw new DOMException_js_1.default(`Failed to execute 'removeProperty' on 'CSSStyleDeclaration': These styles are computed, and therefore the '${name}' property is read-only.`, DOMExceptionNameEnum_js_1.default.domException);
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
exports.default = AbstractCSSStyleDeclaration;
//# sourceMappingURL=AbstractCSSStyleDeclaration.cjs.map