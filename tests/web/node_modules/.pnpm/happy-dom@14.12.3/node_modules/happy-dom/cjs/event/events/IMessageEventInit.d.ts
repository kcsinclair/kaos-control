import IEventInit from '../IEventInit.cjs';
import BrowserWindow from '../../window/BrowserWindow.cjs';
import IMessagePort from '../IMessagePort.cjs';
export default interface IMessageEventInit extends IEventInit {
    data?: unknown | null;
    origin?: string;
    lastEventId?: string;
    source?: BrowserWindow | null;
    ports?: IMessagePort[];
}
//# sourceMappingURL=IMessageEventInit.d.ts.map