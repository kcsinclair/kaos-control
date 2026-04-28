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
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const SelectorCombinatorEnum_js_1 = __importDefault(require("./SelectorCombinatorEnum.cjs"));
/**
 * Selector item.
 */
class SelectorItem {
    /**
     * Constructor.
     *
     * @param [options] Options.
     * @param [options.combinator] Combinator.
     * @param [options.tagName] Tag name.
     * @param [options.id] ID.
     * @param [options.classNames] Class names.
     * @param [options.attributes] Attributes.
     * @param [options.pseudos] Pseudos.
     * @param [options.isPseudoElement] Is pseudo element.
     * @param [options.ignoreErrors] Ignore errors.
     */
    constructor(options) {
        this.tagName = options?.tagName || null;
        this.id = options?.id || null;
        this.classNames = options?.classNames || null;
        this.attributes = options?.attributes || null;
        this.pseudos = options?.pseudos || null;
        this.isPseudoElement = options?.isPseudoElement || false;
        this.combinator = options?.combinator || SelectorCombinatorEnum_js_1.default.descendant;
        this.ignoreErrors = options?.ignoreErrors || false;
    }
    /**
     * Matches a selector against an element.
     *
     * @param element HTML element.
     * @returns Result.
     */
    match(element) {
        let priorityWeight = 0;
        if (this.isPseudoElement) {
            return null;
        }
        // Tag name match
        if (this.tagName) {
            if (this.tagName !== '*' && this.tagName !== element[PropertySymbol.tagName].toUpperCase()) {
                return null;
            }
            priorityWeight += 1;
        }
        // ID Match
        if (this.id) {
            if (this.id !== element.id) {
                return null;
            }
            priorityWeight += 100;
        }
        // Class match
        if (this.classNames) {
            const result = this.matchClass(element);
            if (!result) {
                return null;
            }
            priorityWeight += result.priorityWeight;
        }
        // Attribute match
        if (this.attributes) {
            const result = this.matchAttributes(element);
            if (!result) {
                return null;
            }
            priorityWeight += result.priorityWeight;
        }
        // Pseudo match
        if (this.pseudos) {
            const result = this.matchPseudo(element);
            if (!result) {
                return null;
            }
            priorityWeight += result.priorityWeight;
        }
        return { priorityWeight };
    }
    /**
     * Matches a pseudo selector.
     *
     * @param element Element.
     * @returns Result.
     */
    matchPseudo(element) {
        const parent = element[PropertySymbol.parentNode];
        const parentChildren = element[PropertySymbol.parentNode]
            ? element[PropertySymbol.parentNode][PropertySymbol.children]
            : [];
        if (!this.pseudos) {
            return { priorityWeight: 0 };
        }
        let priorityWeight = 0;
        for (const pseudo of this.pseudos) {
            // Validation
            switch (pseudo.name) {
                case 'not':
                case 'nth-child':
                case 'nth-of-type':
                case 'nth-last-child':
                case 'nth-last-of-type':
                case 'is':
                case 'where':
                    if (!pseudo.arguments) {
                        if (this.ignoreErrors) {
                            return null;
                        }
                        throw new DOMException_js_1.default(`Failed to execute 'matches' on '${element.constructor.name}': '${this.getSelectorString()}' is not a valid selector.`);
                    }
                    break;
            }
            // Check if parent exists
            if (!parent) {
                switch (pseudo.name) {
                    case 'first-child':
                    case 'last-child':
                    case 'only-child':
                    case 'first-of-type':
                    case 'last-of-type':
                    case 'only-of-type':
                    case 'nth-child':
                    case 'nth-of-type':
                    case 'nth-last-child':
                    case 'nth-last-of-type':
                        return null;
                }
            }
            const selectorMatch = this.matchPseudoItem(element, parentChildren, pseudo);
            if (!selectorMatch) {
                return null;
            }
            priorityWeight += selectorMatch.priorityWeight;
        }
        return { priorityWeight };
    }
    /**
     * Matches a pseudo selector.
     *
     * @param element Element.
     * @param parentChildren Parent children.
     * @param pseudo Pseudo.
     */
    matchPseudoItem(element, parentChildren, pseudo) {
        switch (pseudo.name) {
            case 'first-child':
                return parentChildren[0] === element ? { priorityWeight: 10 } : null;
            case 'last-child':
                return parentChildren.length && parentChildren[parentChildren.length - 1] === element
                    ? { priorityWeight: 10 }
                    : null;
            case 'only-child':
                return parentChildren.length === 1 && parentChildren[0] === element
                    ? { priorityWeight: 10 }
                    : null;
            case 'first-of-type':
                for (const child of parentChildren) {
                    if (child[PropertySymbol.tagName] === element[PropertySymbol.tagName]) {
                        return child === element ? { priorityWeight: 10 } : null;
                    }
                }
                return null;
            case 'last-of-type':
                for (let i = parentChildren.length - 1; i >= 0; i--) {
                    const child = parentChildren[i];
                    if (child[PropertySymbol.tagName] === element[PropertySymbol.tagName]) {
                        return child === element ? { priorityWeight: 10 } : null;
                    }
                }
                return null;
            case 'only-of-type':
                let isFound = false;
                for (const child of parentChildren) {
                    if (child[PropertySymbol.tagName] === element[PropertySymbol.tagName]) {
                        if (isFound || child !== element) {
                            return null;
                        }
                        isFound = true;
                    }
                }
                return isFound ? { priorityWeight: 10 } : null;
            case 'checked':
                return element[PropertySymbol.tagName] === 'INPUT' && element.checked
                    ? { priorityWeight: 10 }
                    : null;
            case 'empty':
                return !element[PropertySymbol.children].length ? { priorityWeight: 10 } : null;
            case 'root':
                return element[PropertySymbol.tagName] === 'HTML' ? { priorityWeight: 10 } : null;
            case 'not':
                return !pseudo.selectorItems[0].match(element) ? { priorityWeight: 10 } : null;
            case 'nth-child':
                const nthChildIndex = pseudo.selectorItems[0]
                    ? parentChildren.filter((child) => pseudo.selectorItems[0].match(child)).indexOf(element)
                    : parentChildren.indexOf(element);
                return nthChildIndex !== -1 && pseudo.nthFunction(nthChildIndex + 1)
                    ? { priorityWeight: 10 }
                    : null;
            case 'nth-of-type':
                if (!element[PropertySymbol.parentNode]) {
                    return null;
                }
                const nthOfTypeIndex = parentChildren
                    .filter((child) => child[PropertySymbol.tagName] === element[PropertySymbol.tagName])
                    .indexOf(element);
                return nthOfTypeIndex !== -1 && pseudo.nthFunction(nthOfTypeIndex + 1)
                    ? { priorityWeight: 10 }
                    : null;
            case 'nth-last-child':
                const nthLastChildIndex = pseudo.selectorItems[0]
                    ? parentChildren
                        .filter((child) => pseudo.selectorItems[0].match(child))
                        .reverse()
                        .indexOf(element)
                    : parentChildren.reverse().indexOf(element);
                return nthLastChildIndex !== -1 && pseudo.nthFunction(nthLastChildIndex + 1)
                    ? { priorityWeight: 10 }
                    : null;
            case 'nth-last-of-type':
                const nthLastOfTypeIndex = parentChildren
                    .filter((child) => child[PropertySymbol.tagName] === element[PropertySymbol.tagName])
                    .reverse()
                    .indexOf(element);
                return nthLastOfTypeIndex !== -1 && pseudo.nthFunction(nthLastOfTypeIndex + 1)
                    ? { priorityWeight: 10 }
                    : null;
            case 'target':
                const hash = element[PropertySymbol.ownerDocument].location.hash;
                if (!hash) {
                    return null;
                }
                return element.isConnected && element.id === hash.slice(1) ? { priorityWeight: 10 } : null;
            case 'is':
                let priorityWeight = 0;
                for (const selectorItem of pseudo.selectorItems) {
                    const match = selectorItem.match(element);
                    if (match) {
                        priorityWeight = match.priorityWeight;
                    }
                }
                return priorityWeight ? { priorityWeight } : null;
            case 'where':
                for (const selectorItem of pseudo.selectorItems) {
                    if (selectorItem.match(element)) {
                        return { priorityWeight: 0 };
                    }
                }
                return null;
            default:
                return null;
        }
    }
    /**
     * Matches attribute.
     *
     * @param element Element.
     * @returns Result.
     */
    matchAttributes(element) {
        if (!this.attributes) {
            return null;
        }
        let priorityWeight = 0;
        for (const attribute of this.attributes) {
            const elementAttribute = element[PropertySymbol.attributes].getNamedItem(attribute.name);
            if (!elementAttribute) {
                return null;
            }
            priorityWeight += 10;
            if (attribute.value !== null &&
                (elementAttribute[PropertySymbol.value] === null ||
                    (attribute.regExp && !attribute.regExp.test(elementAttribute[PropertySymbol.value])) ||
                    (!attribute.regExp && attribute.value !== elementAttribute[PropertySymbol.value]))) {
                return null;
            }
        }
        return { priorityWeight };
    }
    /**
     * Matches class.
     *
     * @param element Element.
     * @returns Result.
     */
    matchClass(element) {
        if (!this.classNames) {
            return null;
        }
        const classList = element.className.split(' ');
        let priorityWeight = 0;
        for (const className of this.classNames) {
            if (!classList.includes(className)) {
                return null;
            }
            priorityWeight += 10;
        }
        return { priorityWeight };
    }
    /**
     * Returns the selector string.
     *
     * @returns Selector string.
     */
    getSelectorString() {
        return `${this.tagName ? this.tagName.toLowerCase() : ''}${this.id ? `#${this.id}` : ''}${this.classNames ? `.${this.classNames.join('.')}` : ''}${this.attributes
            ? this.attributes
                .map((attribute) => `[${attribute.name}${attribute.value ? `${attribute.operator || ''}="${attribute.value}"` : ''}]`)
                .join('')
            : ''}${this.pseudos
            ? this.pseudos
                .map((pseudo) => `:${pseudo.name}${pseudo.arguments ? `(${pseudo.arguments})` : ''}`)
                .join('')
            : ''}`;
    }
}
exports.default = SelectorItem;
//# sourceMappingURL=SelectorItem.cjs.map