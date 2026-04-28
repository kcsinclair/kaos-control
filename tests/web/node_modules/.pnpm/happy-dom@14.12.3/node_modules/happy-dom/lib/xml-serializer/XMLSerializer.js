import * as PropertySymbol from '../PropertySymbol.js';
import Node from '../nodes/node/Node.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';
import * as Entities from 'entities';
import HTMLElementConfig from '../config/HTMLElementConfig.js';
import HTMLElementConfigContentModelEnum from '../config/HTMLElementConfigContentModelEnum.js';
/**
 * Utility for converting an element to string.
 */
export default class XMLSerializer {
    /**
     * Constructor.
     *
     * @param [options] Options.
     * @param [options.includeShadowRoots] Include shadow roots.
     * @param [options.escapeEntities] Escape text.
     */
    constructor(options) {
        this.options = {
            includeShadowRoots: false,
            escapeEntities: true
        };
        if (options) {
            if (options.includeShadowRoots !== undefined) {
                this.options.includeShadowRoots = options.includeShadowRoots;
            }
            if (options.escapeEntities !== undefined) {
                this.options.escapeEntities = options.escapeEntities;
            }
        }
    }
    /**
     * Renders an element as HTML.
     *
     * @param root Root element.
     * @returns Result.
     */
    serializeToString(root) {
        switch (root[PropertySymbol.nodeType]) {
            case NodeTypeEnum.elementNode:
                const element = root;
                const localName = element[PropertySymbol.localName];
                const config = HTMLElementConfig[element[PropertySymbol.localName]];
                if (config?.contentModel === HTMLElementConfigContentModelEnum.noDescendants) {
                    return `<${localName}${this.getAttributes(element)}>`;
                }
                const childNodes = localName === 'template'
                    ? root.content[PropertySymbol.childNodes]
                    : root[PropertySymbol.childNodes];
                let innerHTML = '';
                for (const node of childNodes) {
                    innerHTML += this.serializeToString(node);
                }
                // TODO: Should we include closed shadow roots?
                // We are currently only including open shadow roots.
                if (this.options.includeShadowRoots && element.shadowRoot) {
                    innerHTML += `<template shadowrootmode="${element.shadowRoot[PropertySymbol.mode]}">`;
                    for (const node of element.shadowRoot[PropertySymbol.childNodes]) {
                        innerHTML += this.serializeToString(node);
                    }
                    innerHTML += '</template>';
                }
                return `<${localName}${this.getAttributes(element)}>${innerHTML}</${localName}>`;
            case Node.DOCUMENT_FRAGMENT_NODE:
            case Node.DOCUMENT_NODE:
                let html = '';
                for (const node of root[PropertySymbol.childNodes]) {
                    html += this.serializeToString(node);
                }
                return html;
            case NodeTypeEnum.commentNode:
                return `<!--${root.textContent}-->`;
            case NodeTypeEnum.processingInstructionNode:
                // TODO: Add support for processing instructions.
                return `<!--?${root.target} ${root.textContent}?-->`;
            case NodeTypeEnum.textNode:
                return this.options.escapeEntities
                    ? Entities.escapeText(root.textContent)
                    : root.textContent;
            case NodeTypeEnum.documentTypeNode:
                const doctype = root;
                const identifier = doctype.publicId ? ' PUBLIC' : doctype.systemId ? ' SYSTEM' : '';
                const publicId = doctype.publicId ? ` "${doctype.publicId}"` : '';
                const systemId = doctype.systemId ? ` "${doctype.systemId}"` : '';
                return `<!DOCTYPE ${doctype.name}${identifier}${publicId}${systemId}>`;
        }
        return '';
    }
    /**
     * Returns attributes as a string.
     *
     * @param element Element.
     * @returns Attributes.
     */
    getAttributes(element) {
        let attributeString = '';
        if (!element[PropertySymbol.attributes].getNamedItem('is') &&
            element[PropertySymbol.isValue]) {
            attributeString += ' is="' + element[PropertySymbol.isValue] + '"';
        }
        for (let i = 0, max = element[PropertySymbol.attributes].length; i < max; i++) {
            const attribute = element[PropertySymbol.attributes][i];
            if (attribute[PropertySymbol.value] !== null) {
                const escapedValue = this.options.escapeEntities
                    ? Entities.escapeText(attribute[PropertySymbol.value])
                    : attribute[PropertySymbol.value];
                attributeString += ' ' + attribute[PropertySymbol.name] + '="' + escapedValue + '"';
            }
        }
        return attributeString;
    }
}
//# sourceMappingURL=XMLSerializer.js.map