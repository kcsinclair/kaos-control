"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CSSRule_js_1 = __importDefault(require("../CSSRule.cjs"));
/**
 * CSSRule interface.
 */
class CSSContainerRule extends CSSRule_js_1.default {
    constructor() {
        super(...arguments);
        this.type = CSSRule_js_1.default.CONTAINER_RULE;
        this.cssRules = [];
        this.conditionText = '';
    }
    /**
     * Returns css text.
     *
     * @returns CSS text.
     */
    get cssText() {
        let cssText = '';
        for (const cssRule of this.cssRules) {
            cssText += cssRule.cssText;
        }
        return `@container ${this.conditionText} { ${cssText} }`;
    }
}
exports.default = CSSContainerRule;
//# sourceMappingURL=CSSContainerRule.cjs.map