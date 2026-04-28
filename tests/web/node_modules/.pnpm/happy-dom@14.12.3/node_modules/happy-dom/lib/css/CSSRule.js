import CSSRuleTypeEnum from './CSSRuleTypeEnum.js';
/**
 * CSSRule interface.
 */
class CSSRule {
    constructor() {
        this.parentRule = null;
        this.parentStyleSheet = null;
        this.type = null;
    }
    /**
     * Returns selector text.
     *
     * @returns Selector text.
     */
    get cssText() {
        return '';
    }
}
CSSRule.CONTAINER_RULE = CSSRuleTypeEnum.containerRule;
CSSRule.STYLE_RULE = CSSRuleTypeEnum.styleRule;
CSSRule.IMPORT_RULE = CSSRuleTypeEnum.importRule;
CSSRule.MEDIA_RULE = CSSRuleTypeEnum.mediaRule;
CSSRule.FONT_FACE_RULE = CSSRuleTypeEnum.fontFaceRule;
CSSRule.PAGE_RULE = CSSRuleTypeEnum.pageRule;
CSSRule.KEYFRAMES_RULE = CSSRuleTypeEnum.keyframesRule;
CSSRule.KEYFRAME_RULE = CSSRuleTypeEnum.keyframeRule;
CSSRule.NAMESPACE_RULE = CSSRuleTypeEnum.namespaceRule;
CSSRule.COUNTER_STYLE_RULE = CSSRuleTypeEnum.counterStyleRule;
CSSRule.SUPPORTS_RULE = CSSRuleTypeEnum.supportsRule;
CSSRule.DOCUMENT_RULE = CSSRuleTypeEnum.documentRule;
CSSRule.FONT_FEATURE_VALUES_RULE = CSSRuleTypeEnum.fontFeatureValuesRule;
CSSRule.REGION_STYLE_RULE = CSSRuleTypeEnum.regionStyleRule;
export default CSSRule;
//# sourceMappingURL=CSSRule.js.map