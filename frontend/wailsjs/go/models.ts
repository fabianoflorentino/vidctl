export namespace compress {
	
	export class Job {
	    inputPath: string;
	    outputPath: string;
	    presetId: string;
	    sizeMB: number;
	    crf: number;
	    split?: split.Spec;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputPath = source["inputPath"];
	        this.outputPath = source["outputPath"];
	        this.presetId = source["presetId"];
	        this.sizeMB = source["sizeMB"];
	        this.crf = source["crf"];
	        this.split = this.convertValues(source["split"], split.Spec);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class SystemStatus {
	    ffmpegOK: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ffmpegOK = source["ffmpegOK"];
	        this.message = source["message"];
	    }
	}

}

export namespace media {
	
	export class Info {
	    path: string;
	    durationSec: number;
	    width: number;
	    height: number;
	    hasAudio: boolean;
	    hasVideo: boolean;
	    sizeMB: number;
	    videoCodec: string;
	    audioCodec: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.durationSec = source["durationSec"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.hasAudio = source["hasAudio"];
	        this.hasVideo = source["hasVideo"];
	        this.sizeMB = source["sizeMB"];
	        this.videoCodec = source["videoCodec"];
	        this.audioCodec = source["audioCodec"];
	    }
	}

}

export namespace presets {
	
	export class Preset {
	    id: string;
	    name: string;
	    description: string;
	    mode: string;
	    sizeMB: number;
	    crf: number;
	    audioBitrate: string;
	
	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.mode = source["mode"];
	        this.sizeMB = source["sizeMB"];
	        this.crf = source["crf"];
	        this.audioBitrate = source["audioBitrate"];
	    }
	}

}

export namespace split {
	
	export class Spec {
	    parts: number;
	    minutesEach: number;
	
	    static createFrom(source: any = {}) {
	        return new Spec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.parts = source["parts"];
	        this.minutesEach = source["minutesEach"];
	    }
	}

}

export namespace sysinfo {
	
	export class Snapshot {
	    cpu: number;
	    memUsedMB: number;
	    memTotalMB: number;
	    ffmpegCpu: number;
	    gpu: number;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpu = source["cpu"];
	        this.memUsedMB = source["memUsedMB"];
	        this.memTotalMB = source["memTotalMB"];
	        this.ffmpegCpu = source["ffmpegCpu"];
	        this.gpu = source["gpu"];
	    }
	}

}

