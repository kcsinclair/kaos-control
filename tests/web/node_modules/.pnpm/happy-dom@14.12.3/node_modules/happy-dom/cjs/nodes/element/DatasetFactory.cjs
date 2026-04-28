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
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const DatasetUtility_js_1 = __importDefault(require("./DatasetUtility.cjs"));
/**
 * Dataset factory.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/dataset
 */
class DatasetFactory {
    /**
     * @param element The parent element.
     */
    static createDataset(element) {
        // Build the initial dataset record from all data attributes.
        const dataset = {};
        for (let i = 0, max = element[PropertySymbol.attributes].length; i < max; i++) {
            const attribute = element[PropertySymbol.attributes][i];
            if (attribute[PropertySymbol.name].startsWith('data-')) {
                const key = DatasetUtility_js_1.default.kebabToCamelCase(attribute[PropertySymbol.name].replace('data-', ''));
                dataset[key] = attribute[PropertySymbol.value];
            }
        }
        // Documentation for Proxy:
        // https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Proxy
        return new Proxy(dataset, {
            get(dataset, key) {
                const attribute = element[PropertySymbol.attributes].getNamedItem('data-' + DatasetUtility_js_1.default.camelCaseToKebab(key));
                if (attribute) {
                    return (dataset[key] = attribute[PropertySymbol.value]);
                }
                delete dataset[key];
                return undefined;
            },
            set(dataset, key, value) {
                element.setAttribute('data-' + DatasetUtility_js_1.default.camelCaseToKebab(key), value);
                dataset[key] = value;
                return true;
            },
            deleteProperty(dataset, key) {
                element[PropertySymbol.attributes][PropertySymbol.removeNamedItem]('data-' + DatasetUtility_js_1.default.camelCaseToKebab(key));
                return delete dataset[key];
            },
            ownKeys(dataset) {
                // According to Mozilla we have to update the dataset object (target) to contain the same keys as what we return:
                // https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Proxy/Proxy/ownKeys
                // "The result List must contain the keys of all non-configurable own properties of the target object."
                const keys = [];
                const deleteKeys = [];
                for (let i = 0, max = element[PropertySymbol.attributes].length; i < max; i++) {
                    const attribute = element[PropertySymbol.attributes][i];
                    if (attribute[PropertySymbol.name].startsWith('data-')) {
                        const key = DatasetUtility_js_1.default.kebabToCamelCase(attribute[PropertySymbol.name].replace('data-', ''));
                        keys.push(key);
                        dataset[key] = attribute[PropertySymbol.value];
                        if (!dataset[key]) {
                            deleteKeys.push(key);
                        }
                    }
                }
                for (const key of deleteKeys) {
                    delete dataset[key];
                }
                return keys;
            },
            has(_dataset, key) {
                return !!element[PropertySymbol.attributes].getNamedItem('data-' + DatasetUtility_js_1.default.camelCaseToKebab(key));
            }
        });
    }
}
exports.default = DatasetFactory;
//# sourceMappingURL=DatasetFactory.cjs.map