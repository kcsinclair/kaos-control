import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTML Slot Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLSlotElement.
 */
export default class HTMLSlotElement extends HTMLElement {
    constructor() {
        super(...arguments);
        // Events
        this.onslotchange = null;
    }
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this.getAttribute('name') || '';
    }
    /**
     * Sets name.
     *
     * @param name Name.
     */
    set name(name) {
        this.setAttribute('name', name);
    }
    /**
     * Sets the slot's manually assigned nodes to an ordered set of slottables.
     *
     * @param _nodes Nodes.
     */
    assign(..._nodes) {
        // TODO: Do nothing for now. We need to find an example of how it is expected to work before it can be implemented.
    }
    /**
     * Returns assigned nodes.
     *
     * @param [options] Options.
     * @param [options.flatten] A boolean value indicating whether to return the assigned nodes of any available child <slot> elements (true) or not (false). Defaults to false.
     * @returns Nodes.
     */
    assignedNodes(options) {
        const host = this.getRootNode()?.host;
        // TODO: Add support for options.flatten. We need to find an example of how it is expected to work before it can be implemented.
        if (host) {
            const name = this.name;
            if (name) {
                return this.assignedElements(options);
            }
            return host[PropertySymbol.childNodes].slice();
        }
        return [];
    }
    /**
     * Returns assigned elements.
     *
     * @param [_options] Options.
     * @param [_options.flatten] A boolean value indicating whether to return the assigned elements of any available child <slot> elements (true) or not (false). Defaults to false.
     * @returns Nodes.
     */
    assignedElements(_options) {
        const host = this.getRootNode()?.host;
        // TODO: Add support for options.flatten. We need to find an example of how it expected to work before it can be implemented.
        if (host) {
            const name = this.name;
            if (name) {
                const assignedElements = [];
                for (const child of host[PropertySymbol.children]) {
                    if (child.slot === name) {
                        assignedElements.push(child);
                    }
                }
                return assignedElements;
            }
            return host[PropertySymbol.children].slice();
        }
        return [];
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
}
//# sourceMappingURL=HTMLSlotElement.js.map