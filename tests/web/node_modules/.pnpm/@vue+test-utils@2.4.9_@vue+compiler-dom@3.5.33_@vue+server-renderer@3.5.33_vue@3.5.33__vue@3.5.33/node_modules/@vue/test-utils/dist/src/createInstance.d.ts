import type { DefineComponent } from 'vue';
import type { MountingOptions } from './types';
export declare function createInstance(inputComponent: DefineComponent<{}, {}, any, any, any, any>, options?: MountingOptions<any> & Record<string, any>): {
    app: import("vue").App<Element>;
    props: Record<string, unknown>;
    componentRef: import("vue").Ref<null, null>;
};
