export namespace compress {
	
	export class Job {
	    inputPath: string;
	    outputPath: string;
	    presetId: string;
	    sizeMB: number;
	    crf: number;
	
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

