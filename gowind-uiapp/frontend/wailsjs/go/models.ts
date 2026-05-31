export namespace ai {
	
	export class AIProviderPreset {
	    name: string;
	    value: string;
	    baseUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new AIProviderPreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.value = source["value"];
	        this.baseUrl = source["baseUrl"];
	    }
	}
	export class Config {
	    provider: string;
	    baseUrl: string;
	    apiKey: string;
	    model: string;
	    temperature: number;
	    maxTokens: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.baseUrl = source["baseUrl"];
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.temperature = source["temperature"];
	        this.maxTokens = source["maxTokens"];
	    }
	}
	export class MicroservicePartition {
	    serviceName: string;
	    tables: string[];
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new MicroservicePartition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serviceName = source["serviceName"];
	        this.tables = source["tables"];
	        this.description = source["description"];
	    }
	}
	export class OpenAPIResult {
	    success: boolean;
	    files: string[];
	    message?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenAPIResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.files = source["files"];
	        this.message = source["message"];
	        this.error = source["error"];
	    }
	}
	export class PartitionResult {
	    success: boolean;
	    partitions: MicroservicePartition[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new PartitionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.partitions = this.convertValues(source["partitions"], MicroservicePartition);
	        this.error = source["error"];
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
	export class StepResult {
	    success: boolean;
	    content: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new StepResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.content = source["content"];
	        this.error = source["error"];
	    }
	}

}

export namespace configexporter {
	
	export class ExportResult {
	    success: boolean;
	    error?: string;
	    service?: string;
	    filesCount?: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.service = source["service"];
	        this.filesCount = source["filesCount"];
	    }
	}
	export class RemoteConfig {
	    type: string;
	    endpoint: string;
	    projectName: string;
	    group: string;
	    env: string;
	    namespaceId: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoteConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.endpoint = source["endpoint"];
	        this.projectName = source["projectName"];
	        this.group = source["group"];
	        this.env = source["env"];
	        this.namespaceId = source["namespaceId"];
	    }
	}
	export class ServiceInfo {
	    name: string;
	    configFiles: string[];
	    configFolder: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.configFiles = source["configFiles"];
	        this.configFolder = source["configFolder"];
	    }
	}

}

export namespace database {
	
	export class ColumnInfo {
	    name: string;
	    type: string;
	    nullable: boolean;
	    primaryKey: boolean;
	    default: string;
	    comment: string;
	    extra: string;
	
	    static createFrom(source: any = {}) {
	        return new ColumnInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.nullable = source["nullable"];
	        this.primaryKey = source["primaryKey"];
	        this.default = source["default"];
	        this.comment = source["comment"];
	        this.extra = source["extra"];
	    }
	}
	export class DBError {
	    code: string;
	    message: string;
	    details: string;
	
	    static createFrom(source: any = {}) {
	        return new DBError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	        this.details = source["details"];
	    }
	}
	export class ConnectionResult {
	    success: boolean;
	    message: string;
	    database: string;
	    serverVer: string;
	    duration: number;
	    tables: number;
	    connected: boolean;
	    error?: DBError;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.database = source["database"];
	        this.serverVer = source["serverVer"];
	        this.duration = source["duration"];
	        this.tables = source["tables"];
	        this.connected = source["connected"];
	        this.error = this.convertValues(source["error"], DBError);
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
	export class DBConfig {
	    type: string;
	    host: string;
	    port: number;
	    database: string;
	    username: string;
	    password: string;
	    ssl: boolean;
	    dbPath: string;
	    useDSN?: boolean;
	    dsn?: string;
	    sqlContent?: string;
	    timeout?: number;
	    maxOpenConns?: number;
	
	    static createFrom(source: any = {}) {
	        return new DBConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.database = source["database"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.ssl = source["ssl"];
	        this.dbPath = source["dbPath"];
	        this.useDSN = source["useDSN"];
	        this.dsn = source["dsn"];
	        this.sqlContent = source["sqlContent"];
	        this.timeout = source["timeout"];
	        this.maxOpenConns = source["maxOpenConns"];
	    }
	}
	
	export class TableInfo {
	    table_name: string;
	    table_type: string;
	    table_engine: string;
	    table_rows: number;
	    table_comment: string;
	    table_columns: number;
	    table_indexes: number;
	    create_time: string;
	
	    static createFrom(source: any = {}) {
	        return new TableInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table_name = source["table_name"];
	        this.table_type = source["table_type"];
	        this.table_engine = source["table_engine"];
	        this.table_rows = source["table_rows"];
	        this.table_comment = source["table_comment"];
	        this.table_columns = source["table_columns"];
	        this.table_indexes = source["table_indexes"];
	        this.create_time = source["create_time"];
	    }
	}

}

export namespace detect {
	
	export class Module {
	    Path: string;
	    Version: string;
	
	    static createFrom(source: any = {}) {
	        return new Module(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Path = source["Path"];
	        this.Version = source["Version"];
	    }
	}
	export class ProjectInfo {
	    Root: string;
	    GoVersion: string;
	    ModPath: string;
	    Main: boolean;
	    Version: string;
	    Replace?: Module;
	    Dependencies?: Module[];
	    Services?: string[];
	    HasApi?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProjectInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Root = source["Root"];
	        this.GoVersion = source["GoVersion"];
	        this.ModPath = source["ModPath"];
	        this.Main = source["Main"];
	        this.Version = source["Version"];
	        this.Replace = this.convertValues(source["Replace"], Module);
	        this.Dependencies = this.convertValues(source["Dependencies"], Module);
	        this.Services = source["Services"];
	        this.HasApi = source["HasApi"];
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

export namespace devtools {
	
	export class AddServiceOptions {
	    serviceName: string;
	    servers: string[];
	    dbClients: string[];
	
	    static createFrom(source: any = {}) {
	        return new AddServiceOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serviceName = source["serviceName"];
	        this.servers = source["servers"];
	        this.dbClients = source["dbClients"];
	    }
	}
	export class CommandResult {
	    success: boolean;
	    output: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new CommandResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.output = source["output"];
	        this.error = source["error"];
	    }
	}
	export class CreateProjectOptions {
	    name: string;
	    module: string;
	    repoUrl: string;
	    branch: string;
	    parentDir: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateProjectOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.module = source["module"];
	        this.repoUrl = source["repoUrl"];
	        this.branch = source["branch"];
	        this.parentDir = source["parentDir"];
	    }
	}
	export class ServiceInfo {
	    name: string;
	    hasServer: boolean;
	    hasConfig: boolean;
	    hasEnt: boolean;
	    entSchemas?: string[];
	    hasGorm: boolean;
	    gormModels?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ServiceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.hasServer = source["hasServer"];
	        this.hasConfig = source["hasConfig"];
	        this.hasEnt = source["hasEnt"];
	        this.entSchemas = source["entSchemas"];
	        this.hasGorm = source["hasGorm"];
	        this.gormModels = source["gormModels"];
	    }
	}

}

export namespace generator {
	
	export class Option {
	    id: number;
	    tableName: string;
	    service: string;
	    exclude: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Option(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tableName = source["tableName"];
	        this.service = source["service"];
	        this.exclude = source["exclude"];
	    }
	}

}

