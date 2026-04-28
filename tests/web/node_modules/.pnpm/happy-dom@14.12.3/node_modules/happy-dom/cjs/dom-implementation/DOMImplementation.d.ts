import DocumentType from '../nodes/document-type/DocumentType.cjs';
import Document from '../nodes/document/Document.cjs';
/**
 * The DOMImplementation interface represents an object providing methods which are not dependent on any particular document. Such an object is returned by the.
 */
export default class DOMImplementation {
    #private;
    /**
     * Constructor.
     *
     * @param window Window.
     */
    constructor(window: Document);
    /**
     * Creates and returns an XML Document.
     *
     * TODO: Not fully implemented.
     */
    createDocument(): Document;
    /**
     * Creates and returns an HTML Document.
     */
    createHTMLDocument(): Document;
    /**
     * Creates and returns an HTML Document.
     *
     * @param qualifiedName Qualified name.
     * @param publicId Public ID.
     * @param systemId System ID.
     */
    createDocumentType(qualifiedName: string, publicId: string, systemId: string): DocumentType;
}
//# sourceMappingURL=DOMImplementation.d.ts.map