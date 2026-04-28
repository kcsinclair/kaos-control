import NamespaceURI from '../../config/NamespaceURI.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import MutationRecord from '../../mutation-observer/MutationRecord.js';
import MutationTypeEnum from '../../mutation-observer/MutationTypeEnum.js';
import NamedNodeMap from '../../named-node-map/NamedNodeMap.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class ElementNamedNodeMap extends NamedNodeMap {
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
                    observer.report(new MutationRecord({
                        target: this[PropertySymbol.ownerElement],
                        type: MutationTypeEnum.attributes,
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
                    observer.report(new MutationRecord({
                        target: this[PropertySymbol.ownerElement],
                        type: MutationTypeEnum.attributes,
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
        if (this[PropertySymbol.ownerElement][PropertySymbol.namespaceURI] === NamespaceURI.svg) {
            return name;
        }
        return name.toLowerCase();
    }
}
//# sourceMappingURL=ElementNamedNodeMap.js.map