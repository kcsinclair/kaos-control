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
const NamespaceURI_js_1 = __importDefault(require("../../config/NamespaceURI.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const MutationRecord_js_1 = __importDefault(require("../../mutation-observer/MutationRecord.cjs"));
const MutationTypeEnum_js_1 = __importDefault(require("../../mutation-observer/MutationTypeEnum.cjs"));
const NamedNodeMap_js_1 = __importDefault(require("../../named-node-map/NamedNodeMap.cjs"));
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
class ElementNamedNodeMap extends NamedNodeMap_js_1.default {
    /**
     * Constructor.
     *
     * @param ownerElement Owner element.
     */
    constructor(ownerElement) {
        super();
        this[PropertySymbol.ownerElement] = ownerElement;
    }
    /**
     * @override
     */
    getNamedItem(name) {
        return this[PropertySymbol.namedItems][this[PropertySymbol.getAttributeName](name)] || null;
    }
    /**
     * @override
     */
    getNamedItemNS(namespace, localName) {
        return super.getNamedItemNS(namespace, this[PropertySymbol.getAttributeName](localName));
    }
    /**
     * @override
     */
    setNamedItem(item) {
        if (!item[PropertySymbol.name]) {
            return null;
        }
        item[PropertySymbol.name] = this[PropertySymbol.getAttributeName](item[PropertySymbol.name]);
        item[PropertySymbol.ownerElement] = this[PropertySymbol.ownerElement];
        const replacedItem = super.setNamedItem(item);
        const oldValue = replacedItem ? replacedItem[PropertySymbol.value] : null;
        if (this[PropertySymbol.ownerElement][PropertySymbol.isConnected]) {
            this[PropertySymbol.ownerElement].ownerDocument[PropertySymbol.cacheID]++;
        }
        if (item[PropertySymbol.name] === 'class' &&
            this[PropertySymbol.ownerElement][PropertySymbol.classList]) {
            this[PropertySymbol.ownerElement][PropertySymbol.classList][PropertySymbol.updateIndices]();
        }
        if (item[PropertySymbol.name] === 'id' || item[PropertySymbol.name] === 'name') {
            if (this[PropertySymbol.ownerElement][PropertySymbol.parentNode] &&
                this[PropertySymbol.ownerElement][PropertySymbol.parentNode][PropertySymbol.children] &&
                item[PropertySymbol.value] !== oldValue) {
                if (oldValue) {
                    (this[PropertySymbol.ownerElement][PropertySymbol.parentNode][PropertySymbol.children])[PropertySymbol.removeNamedItem](this[PropertySymbol.ownerElement], oldValue);
                }
                if (item[PropertySymbol.value]) {
                    (this[PropertySymbol.ownerElement][PropertySymbol.parentNode][PropertySymbol.children])[PropertySymbol.appendNamedItem](this[PropertySymbol.ownerElement], item[PropertySymbol.value]);
                }
            }
        }
        if (this[PropertySymbol.ownerElement].attributeChangedCallback &&
            this[PropertySymbol.ownerElement].constructor[PropertySymbol.observedAttributes] &&
            this[PropertySymbol.ownerElement].constructor[PropertySymbol.observedAttributes].includes(item[PropertySymbol.name])) {
            this[PropertySymbol.ownerElement].attributeChangedCallback(item[PropertySymbol.name], oldValue, item[PropertySymbol.value]);
        }
        // MutationObserver
        if (this[PropertySymbol.ownerElement][PropertySymbol.observers].length > 0) {
            for (const observer of (this[PropertySymbol.ownerElement][PropertySymbol.observers])) {
                if (observer.options?.attributes &&
                    (!observer.options.attributeFilter ||
                        observer.options.attributeFilter.includes(item[PropertySymbol.name]))) {
                    observer.report(new MutationRecord_js_1.default({
                        target: this[PropertySymbol.ownerElement],
                        type: MutationTypeEnum_js_1.default.attributes,
                        attributeName: item[PropertySymbol.name],
                        oldValue: observer.options.attributeOldValue ? oldValue : null
                    }));
                }
            }
        }
        return replacedItem || null;
    }
    /**
     * @override
     */
    [(PropertySymbol.ownerElement, PropertySymbol.removeNamedItem)](name) {
        const removedItem = super[PropertySymbol.removeNamedItem](this[PropertySymbol.getAttributeName](name));
        if (!removedItem) {
            return null;
        }
        if (this[PropertySymbol.ownerElement][PropertySymbol.isConnected]) {
            this[PropertySymbol.ownerElement].ownerDocument[PropertySymbol.cacheID]++;
        }
        if (removedItem[PropertySymbol.name] === 'class' &&
            this[PropertySymbol.ownerElement][PropertySymbol.classList]) {
            this[PropertySymbol.ownerElement][PropertySymbol.classList][PropertySymbol.updateIndices]();
        }
        if (removedItem[PropertySymbol.name] === 'id' || removedItem[PropertySymbol.name] === 'name') {
            if (this[PropertySymbol.ownerElement][PropertySymbol.parentNode] &&
                this[PropertySymbol.ownerElement][PropertySymbol.parentNode][PropertySymbol.children] &&
                removedItem[PropertySymbol.value]) {
                (this[PropertySymbol.ownerElement][PropertySymbol.parentNode][PropertySymbol.children])[PropertySymbol.removeNamedItem](this[PropertySymbol.ownerElement], removedItem[PropertySymbol.value]);
            }
        }
        if (this[PropertySymbol.ownerElement].attributeChangedCallback &&
            this[PropertySymbol.ownerElement].constructor[PropertySymbol.observedAttributes] &&
            this[PropertySymbol.ownerElement].constructor[PropertySymbol.observedAttributes].includes(removedItem[PropertySymbol.name])) {
            this[PropertySymbol.ownerElement].attributeChangedCallback(removedItem[PropertySymbol.name], removedItem[PropertySymbol.value], null);
        }
        // MutationObserver
        if (this[PropertySymbol.ownerElement][PropertySymbol.observers].length > 0) {
            for (const observer of (this[PropertySymbol.ownerElement][PropertySymbol.observers])) {
                if (observer.options?.attributes &&
                    (!observer.options.attributeFilter ||
                        observer.options.attributeFilter.includes(removedItem[PropertySymbol.name]))) {
                    observer.report(new MutationRecord_js_1.default({
                        target: this[PropertySymbol.ownerElement],
                        type: MutationTypeEnum_js_1.default.attributes,
                        attributeName: removedItem[PropertySymbol.name],
                        oldValue: observer.options.attributeOldValue
                            ? removedItem[PropertySymbol.value]
                            : null
                    }));
                }
            }
        }
        return removedItem;
    }
    /**
     * @override
     */
    removeNamedItemNS(namespace, localName) {
        return super.removeNamedItemNS(namespace, this[PropertySymbol.getAttributeName](localName));
    }
    /**
     * Returns attribute name.
     *
     * @param name Name.
     * @returns Attribute name based on namespace.
     */
    [PropertySymbol.getAttributeName](name) {
        if (this[PropertySymbol.ownerElement][PropertySymbol.namespaceURI] === NamespaceURI_js_1.default.svg) {
            return name;
        }
        return name.toLowerCase();
    }
}
exports.default = ElementNamedNodeMap;
//# sourceMappingURL=ElementNamedNodeMap.cjs.map