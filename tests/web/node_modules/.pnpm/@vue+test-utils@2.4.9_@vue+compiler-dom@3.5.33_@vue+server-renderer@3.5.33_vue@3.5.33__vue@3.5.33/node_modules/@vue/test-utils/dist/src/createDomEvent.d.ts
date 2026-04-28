import type { DomEventNameWithModifier } from './constants/dom-events';
import { KeyName, keyCodesByKeyName } from './constants/dom-events';
interface TriggerOptions {
    code?: string;
    key?: string;
    keyCode?: number;
    [custom: string]: any;
}
declare function createDOMEvent(eventString: DomEventNameWithModifier | string, options?: TriggerOptions): Event & TriggerOptions;
export { TriggerOptions, createDOMEvent, keyCodesByKeyName, KeyName };
