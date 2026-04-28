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
class HTMLOptionElementNamedNodeMap extends HTMLElementNamedNodeMap_js_1.default {
    /**
     * @override
     */
    setNamedItem(item) {
        const replacedItem = super.setNamedItem(item);
        if (!this[PropertySymbol.ownerElement][PropertySymbol.dirtyness] &&
            item[PropertySymbol.name] === 'selected' &&
            replacedItem?.[PropertySymbol.value] !== item[PropertySymbol.value]) {
            const selectNode = (this[PropertySymbol.ownerElement][PropertySymbol.selectNode]);
            this[PropertySymbol.ownerElement][PropertySymbol.selectedness] = true;
            if (selectNode) {
                selectNode[PropertySymbol.updateOptionItems](this[PropertySymbol.ownerElement]);
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
            !this[PropertySymbol.ownerElement][PropertySymbol.dirtyness] &&
            removedItem[PropertySymbol.name] === 'selected') {
            const selectNode = (this[PropertySymbol.ownerElement][PropertySymbol.selectNode]);
            this[PropertySymbol.ownerElement][PropertySymbol.selectedness] = false;
            if (selectNode) {
                selectNode[PropertySymbol.updateOptionItems]();
            }
        }
        return removedItem;
    }
}
exports.default = HTMLOptionElementNamedNodeMap;
//# sourceMappingURL=HTMLOptionElementNamedNodeMap.cjs.map