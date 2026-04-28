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
var _DOMImplementation_document;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const NodeFactory_js_1 = __importDefault(require("../nodes/NodeFactory.cjs"));
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
        const documentType = NodeFactory_js_1.default.createNode(__classPrivateFieldGet(this, _DOMImplementation_document, "f"), __classPrivateFieldGet(this, _DOMImplementation_document, "f")[PropertySymbol.ownerWindow].DocumentType);
        documentType[PropertySymbol.name] = qualifiedName;
        documentType[PropertySymbol.publicId] = publicId;
        documentType[PropertySymbol.systemId] = systemId;
        return documentType;
    }
}
_DOMImplementation_document = new WeakMap();
exports.default = DOMImplementation;
//# sourceMappingURL=DOMImplementation.cjs.map