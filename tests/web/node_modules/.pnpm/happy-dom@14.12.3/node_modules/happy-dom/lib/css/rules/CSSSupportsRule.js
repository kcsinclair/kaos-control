import CSSRule from '../CSSRule.js';
/**
 * CSSRule interface.
 */
export default class CSSSupportsRule extends CSSRule {
    constructor() {
        super(...arguments);
        this.type = CSSRule.SUPPORTS_RULE;
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
        return `@supports ${this.conditionText} { ${cssText} }`;
    }
}
//# sourceMappingURL=CSSSupportsRule.js.map