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
Object.defineProperty(exports, "__esModule", { value: true });
const FocusEvent_js_1 = __importDefault(require("../../event/events/FocusEvent.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * HTMLElement utility.
 */
class HTMLElementUtility {
    /**
     * Triggers a blur event.
     *
     * @param element Element.
     */
    static blur(element) {
        if (element[PropertySymbol.ownerDocument][PropertySymbol.activeElement] !== element ||
            !element[PropertySymbol.isConnected]) {
            return;
        }
        const relatedTarget = element[PropertySymbol.ownerDocument][PropertySymbol.nextActiveElement] ?? null;
        element[PropertySymbol.ownerDocument][PropertySymbol.activeElement] = null;
        element.dispatchEvent(new FocusEvent_js_1.default('blur', {
            relatedTarget,
            bubbles: false,
            composed: true
        }));
        element.dispatchEvent(new FocusEvent_js_1.default('focusout', {
            relatedTarget,
            bubbles: true,
            composed: true
        }));
    }
    /**
     * Triggers a focus event.
     *
     * @param element Element.
     */
    static focus(element) {
        if (element[PropertySymbol.ownerDocument][PropertySymbol.activeElement] === element ||
            !element[PropertySymbol.isConnected]) {
            return;
        }
        // Set the next active element so `blur` can use it for `relatedTarget`.
        element[PropertySymbol.ownerDocument][PropertySymbol.nextActiveElement] = element;
        const relatedTarget = element[PropertySymbol.ownerDocument][PropertySymbol.activeElement];
        if (element[PropertySymbol.ownerDocument][PropertySymbol.activeElement] !== null) {
            element[PropertySymbol.ownerDocument][PropertySymbol.activeElement].blur();
        }
        // Clean up after blur, so it does not affect next blur call.
        element[PropertySymbol.ownerDocument][PropertySymbol.nextActiveElement] = null;
        element[PropertySymbol.ownerDocument][PropertySymbol.activeElement] = element;
        element.dispatchEvent(new FocusEvent_js_1.default('focus', {
            relatedTarget,
            bubbles: false,
            composed: true
        }));
        element.dispatchEvent(new FocusEvent_js_1.default('focusin', {
            relatedTarget,
            bubbles: true,
            composed: true
        }));
    }
}
exports.default = HTMLElementUtility;
//# sourceMappingURL=HTMLElementUtility.cjs.map