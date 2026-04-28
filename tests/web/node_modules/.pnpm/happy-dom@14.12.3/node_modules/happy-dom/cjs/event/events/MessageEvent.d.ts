import BrowserWindow from '../../window/BrowserWindow.cjs';
import Event from '../Event.cjs';
import IMessagePort from '../IMessagePort.cjs';
import IMessageEventInit from './IMessageEventInit.cjs';
/**
 * Message event.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MessageEvent
 */
export default class MessageEvent extends Event {
    readonly data: unknown | null;
    readonly origin: string;
    readonly lastEventId: string;
    readonly source: BrowserWindow | null;
    readonly ports: IMessagePort[];
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IMessageEventInit | null);
}
//# sourceMappingURL=MessageEvent.d.ts.map