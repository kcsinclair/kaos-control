import CSSRule from '../CSSRule.js';
/**
 * CSSRule interface.
 */
export default class CSSContainerRule extends CSSRule {
    constructor() {
        super(...arguments);
        this.type = CSSRule.CONTAINER_RULE;
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
//# sourceMappingURL=CSSContainerRule.js.map