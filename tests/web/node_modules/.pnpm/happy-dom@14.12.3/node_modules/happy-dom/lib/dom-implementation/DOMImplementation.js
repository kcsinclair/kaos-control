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
var _DOMImplementation_document;
import * as PropertySymbol from '../PropertySymbol.js';
import NodeFactory from '../nodes/NodeFactory.js';
/**
 * The DOMImplementation interface represents an object providing methods which are not dependent on any particular document. Such an object is returned by the.
 */
class DOMImplementation {
    /**
     * Constructor.
     *
     * @param window Window.
     */
    constructor(window) {
        _DOMImplementation_document.set(this, void 0);
        __classPrivateFieldSet(this, _DOMImplementation_document, window, "f");
    }
    /**
     * Creates and returns an XML Document.
     *
     * TODO: Not fully implemented.
     */
    createDocument() {
        return new (__classPrivateFieldGet(this, _DOMImplementation_document, "f")[PropertySymbol.ownerWindow].HTMLDocument)();
    }
    /**
     * Creates and returns an HTML Document.
     */
    createHTMLDocument() {
        return new (__classPrivateFieldGet(this, _DOMImplementation_document, "f")[PropertySymbol.ownerWindow].HTMLDocument)();
    }
    /**
     * Creates and returns an HTML Document.
     *
     * @param qualifiedName Qualified name.
     * @param publicId Public ID.
     * @param systemId System ID.
     */
    createDocumentType(qualifiedName, publicId, systemId) {
        const documentType = NodeFactory.createNode(__classPrivateFieldGet(this, _DOMImplementation_document, "f"), __classPrivateFieldGet(this, _DOMImplementation_document, "f")[PropertySymbol.ownerWindow].DocumentType);
        documentType[PropertySymbol.name] = qualifiedName;
        documentType[PropertySymbol.publicId] = publicId;
        documentType[PropertySymbol.systemId] = systemId;
        return documentType;
    }
}
_DOMImplementation_document = new WeakMap();
export default DOMImplementation;
//# sourceMappingURL=DOMImplementation.js.map