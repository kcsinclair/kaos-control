import Event from '../Event.cjs';
import ICustomEventInit from './ICustomEventInit.cjs';
/**
 *
 */
export default class CustomEvent extends Event {
    detail: any;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: ICustomEventInit | null);
    /**
     * Init event.
     *
     * @deprecated
     * @param type Type.
     * @param [bubbles=false] "true" if it bubbles.
     * @param [cancelable=false] "true" if it cancelable.
     * @param [detail=null] Custom event detail.
     */
    initCustomEvent(type: string, bubbles?: boolean, cancelable?: boolean, detail?: object): void;
}
//# sourceMappingURL=CustomEvent.d.ts.map