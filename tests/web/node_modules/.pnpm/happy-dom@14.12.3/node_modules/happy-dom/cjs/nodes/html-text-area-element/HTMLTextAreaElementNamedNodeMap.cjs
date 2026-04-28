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
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const HTMLElementNamedNodeMap_js_1 = __importDefault(require("../html-element/HTMLElementNamedNodeMap.cjs"));
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
class HTMLTextAreaElementNamedNodeMap extends HTMLElementNamedNodeMap_js_1.default {
    /**
     * @override
     */
    setNamedItem(item) {
        const replacedItem = super.setNamedItem(item);
        if ((item[PropertySymbol.name] === 'id' || item[PropertySymbol.name] === 'name') &&
            this[PropertySymbol.ownerElement][PropertySymbol.formNode]) {
            if (replacedItem && replacedItem[PropertySymbol.value]) {
                this[PropertySymbol.ownerElement][PropertySymbol.formNode][PropertySymbol.removeFormControlItem](this[PropertySymbol.ownerElement], replacedItem[PropertySymbol.value]);
            }
            if (item[PropertySymbol.value]) {
                this[PropertySymbol.ownerElement][PropertySymbol.formNode][PropertySymbol.appendFormControlItem](this[PropertySymbol.ownerElement], item[PropertySymbol.value]);
            }
        }
        return replacedItem || null;
    }
    /**
     * @override
     */
    [(PropertySymbol.ownerElement, PropertySymbol.removeNamedItem)](name) {
        const removedItem = super[PropertySymbol.removeNamedItem](name);
        if (removedItem &&
            (removedItem[PropertySymbol.name] === 'id' || removedItem[PropertySymbol.name] === 'name') &&
            this[PropertySymbol.ownerElement][PropertySymbol.formNode]) {
            this[PropertySymbol.ownerElement][PropertySymbol.formNode][PropertySymbol.removeFormControlItem](this[PropertySymbol.ownerElement], removedItem[PropertySymbol.value]);
        }
        return removedItem;
    }
}
exports.default = HTMLTextAreaElementNamedNodeMap;
//# sourceMappingURL=HTMLTextAreaElementNamedNodeMap.cjs.map