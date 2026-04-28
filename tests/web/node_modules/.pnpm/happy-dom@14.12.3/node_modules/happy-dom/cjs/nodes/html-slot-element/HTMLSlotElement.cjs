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
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * HTML Slot Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLSlotElement.
 */
class HTMLSlotElement extends HTMLElement_js_1.default {
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
exports.default = HTMLSlotElement;
//# sourceMappingURL=HTMLSlotElement.cjs.map