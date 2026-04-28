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
const XMLParser_js_1 = __importDefault(require("../../xml-parser/XMLParser.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const HTMLCollection_js_1 = __importDefault(require("../element/HTMLCollection.cjs"));
const NamespaceURI_js_1 = __importDefault(require("../../config/NamespaceURI.cjs"));
/**
 * Parent node utility.
 */
class ParentNodeUtility {
    /**
     * Inserts a set of Node objects or DOMString objects after the last child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param parentNode Parent node.
     * @param nodes List of Node or DOMString.
     */
    static append(parentNode, ...nodes) {
        for (const node of nodes) {
            if (typeof node === 'string') {
                XMLParser_js_1.default.parse(parentNode[PropertySymbol.ownerDocument], node, {
                    rootNode: parentNode
                });
            }
            else {
                parentNode.appendChild(node);
            }
        }
    }
    /**
     * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param parentNode Parent node.
     * @param nodes List of Node or DOMString.
     */
    static prepend(parentNode, ...nodes) {
        const firstChild = parentNode.firstChild;
        for (const node of nodes) {
            if (typeof node === 'string') {
                const newChildNodes = (XMLParser_js_1.default.parse(parentNode[PropertySymbol.ownerDocument], node))[PropertySymbol.childNodes].slice();
                for (const newChildNode of newChildNodes) {
                    parentNode.insertBefore(newChildNode, firstChild);
                }
            }
            else {
                parentNode.insertBefore(node, firstChild);
            }
        }
    }
    /**
     * Replaces the existing children of a ParentNode with a specified new set of children.
     *
     * @param parentNode Parent node.
     * @param nodes List of Node or DOMString.
     */
    static replaceChildren(parentNode, ...nodes) {
        for (const node of parentNode[PropertySymbol.childNodes].slice()) {
            parentNode.removeChild(node);
        }
        this.append(parentNode, ...nodes);
    }
    /**
     * Returns an elements by class name.
     *
     * @param parentNode Parent node.
     * @param className Tag name.
     * @returns Matching element.
     */
    static getElementsByClassName(parentNode, className) {
        let matches = new HTMLCollection_js_1.default();
        for (const child of parentNode[PropertySymbol.children]) {
            if (child.className.split(' ').includes(className)) {
                matches.push(child);
            }
            matches = (matches.concat(this.getElementsByClassName(child, className)));
        }
        return matches;
    }
    /**
     * Returns an elements by tag name.
     *
     * @param parentNode Parent node.
     * @param tagName Tag name.
     * @returns Matching element.
     */
    static getElementsByTagName(parentNode, tagName) {
        const upperTagName = tagName.toUpperCase();
        const includeAll = tagName === '*';
        let matches = new HTMLCollection_js_1.default();
        for (const child of parentNode[PropertySymbol.children]) {
            if (includeAll || child[PropertySymbol.tagName].toUpperCase() === upperTagName) {
                matches.push(child);
            }
            matches = (matches.concat(this.getElementsByTagName(child, tagName)));
        }
        return matches;
    }
    /**
     * Returns an elements by tag name and namespace.
     *
     * @param parentNode Parent node.
     * @param namespaceURI Namespace URI.
     * @param tagName Tag name.
     * @returns Matching element.
     */
    static getElementsByTagNameNS(parentNode, namespaceURI, tagName) {
        // When the namespace is HTML, the tag name is case-insensitive.
        const formattedTagName = namespaceURI === NamespaceURI_js_1.default.html ? tagName.toUpperCase() : tagName;
        const includeAll = tagName === '*';
        let matches = new HTMLCollection_js_1.default();
        for (const child of parentNode[PropertySymbol.children]) {
            if ((includeAll || child[PropertySymbol.tagName] === formattedTagName) &&
                child[PropertySymbol.namespaceURI] === namespaceURI) {
                matches.push(child);
            }
            matches = (matches.concat(this.getElementsByTagNameNS(child, namespaceURI, tagName)));
        }
        return matches;
    }
    /**
     * Returns the first element matching a tag name.
     * This is not part of the browser standard and is only used internally in the document.
     *
     * @param parentNode Parent node.
     * @param tagName Tag name.
     * @returns Matching element.
     */
    static getElementByTagName(parentNode, tagName) {
        const upperTagName = tagName.toUpperCase();
        for (const child of parentNode[PropertySymbol.children]) {
            if (child[PropertySymbol.tagName] === upperTagName) {
                return child;
            }
            const match = this.getElementByTagName(child, tagName);
            if (match) {
                return match;
            }
        }
        return null;
    }
    /**
     * Returns an element by ID.
     *
     * @param parentNode Parent node.
     * @param id ID.
     * @returns Matching element.
     */
    static getElementById(parentNode, id) {
        id = String(id);
        for (const child of parentNode[PropertySymbol.children]) {
            if (child.id === id) {
                return child;
            }
            const match = this.getElementById(child, id);
            if (match) {
                return match;
            }
        }
        return null;
    }
}
exports.default = ParentNodeUtility;
//# sourceMappingURL=ParentNodeUtility.cjs.map