// Type definitions for peek-stream 1.1
// Project: https://github.com/mafintosh/peek-stream
declare module 'peek-stream' {
  import { Transform } from 'stream';

  interface PeekOptions {
    newline?: boolean;
    maxBuffer?: number;
    strict?: boolean;
  }

  type SwapCallback = (error?: Error | null, parser?: Transform) => void;
  type OnPeekCallback = (data: Buffer, swap: SwapCallback) => void;

  function peek(options: PeekOptions, onPeek: OnPeekCallback): Transform;
  function peek(onPeek: OnPeekCallback): Transform;

  export = peek;
}
