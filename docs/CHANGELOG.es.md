# Changelog

**Español · [Português (Brasil)](../CHANGELOG.md) · [English](CHANGELOG.en.md) ·
[中文（简体）](CHANGELOG.zh-CN.md)**

Todos los cambios relevantes de vidctl, en el formato
[Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/), con versionado
[SemVer](https://semver.org/lang/es/) y una release por plataforma según el
workflow de publicación.

Las releases por plataforma (`vX.Y.Z-windows`, `vX.Y.Z-linux`,
`vX.Y.Z-macos`) comparten este mismo changelog: la diferencia entre ellas es
solo el sistema operativo de los paquetes.

## [Unreleased]

- **Notificaciones del sistema** (fase 7): aviso del escritorio cuando una
  tarea termina o falla (`notifyOnDone`) y la opción de revelar la carpeta del
  archivo de salida al finalizar (`openFolderOnDone`), configurables en la
  nueva sección "Notificaciones" de Preferencias. El backend entrega vía
  `beeep` (toast en Windows, Centro de notificaciones en macOS, `notify-send`
  en Linux) y abre la carpeta con el gestor de archivos del sistema; las
  notificaciones nunca afectan el resultado de la conversión.

## [2.4.0] — 2026-10-12

- **i18n (pt-BR/en/es/zh)**: la interfaz ganó diccionarios en portugués
  (Brasil, predeterminado), inglés, español y chino simplificado (mandarín). El
  idioma se elige en Preferencias y queda guardado entre ejecuciones
  (`config.Language`).
- **Sitio y documentación en 4 idiomas**: la landing ganó un selector de idioma
  (PT/EN/ES/中文); `README`, `CHANGELOG` y la guía de instalación tienen
  versiones en pt-BR, en, es y zh-CN. El plan de implementación interno se
  mantuvo en pt (es descartable).
- **Base del código en inglés**: comentarios (estilo go doc), mensajes de
  usuario, logs y rótulos de test se anglicizaron en el backend (Go) y frontend
  (Svelte/TS); los diccionarios de la UI son la única fuente de texto en pt.

## [2.3.0] — 2026-10-11

- **Extraer audio de un vídeo** (fase 6): la nueva sección **"Extraer audio"** de
  la barra lateral toma la pista sin recodificar el vídeo, eligiendo formato
  (mp3, opus, aac, flac, wav o copiar — stream copy, solo remux), calidad (kbps),
  canales (mono/estéreo), normalización EBU R128 (loudnorm) y una ventana de
  tiempo (mm:ss). La extracción entra a la misma cola de tareas, con progreso,
  cancelación y abrir carpeta al terminar. El sufijo del archivo de salida se
  configura en Preferencias (`-audio` por defecto).
- **Nombres de la barra lateral**: "Cortar en partes" pasó a ser **"Cortar"** y
  "Ajustes por archivo" pasó a ser **"Ajustes de vídeo"**, dejando claro qué
  pertenece al audio (extracción) y qué al vídeo (escala, corte, FPS, rotación…)
- **Un vídeo a la vez**: "Agregar vídeos…" pasó a ser **"Cambiar vídeo…"** —
  cambiar reemplaza la selección, **reinicia las opciones por archivo** al
  predeterminado y ya no encola un lote (la app procesa un trabajo a la vez,
  secuencial).
- **Cola de conversiones → Cola de tareas**: la cola ahora lista cualquier
  trabajo (comprimir o extraer audio), con contador de tareas.
- **Menos redundancia**: con un vídeo cargado, el área de destino (drop-target)
  arriba del contenido se eliminó — el cambio queda en el botón **"Cambiar
  vídeo…"** o soltando un archivo nuevo en cualquier parte de la ventana; el área
  solo permanece en la pantalla vacía.
- **Tipografía más grande**: cada tamaño de fuente de la interfaz subió **1px**
  (la jerarquía se mantiene; los textos densos del panel y la cola se leen más
  cómodos).
- **Botón "extraer audio" en la tarjeta del vídeo**: las opciones de audio siguen
  en la sidebar, pero el botón de extraer ahora vive en su propia tarjeta bajo el
  destino del vídeo (misma apariencia del "Guardar como…"), con la ruta del audio
  mostrando la extensión del formato elegido (`.mp3`, `.flac`, `.wav`…) y un
  **"Guardar como…"** para elegir otro destino.
- **Ventana más grande por defecto**: la ventana abre en **1360×768** (antes
  1000×700).
- **"Tamaño objetivo" → "Tamaño"**: la sección de la sidebar quedó solo
  **Tamaño** y la etiqueta del control pasó a **megabytes** (sin la ambigüedad de
  MB = megabit).
- **Tema "sistema" fiel**: la opción de apariencia "Sistema" ahora lee el tema
  real del escritorio (gsettings/GTK en Linux, `AppleInterfaceStyle` en macOS,
  registro en Windows) en vez de confiar en `prefers-color-scheme` del WebKit,
  que en Linux reportaba "claro" incluso con el sistema oscuro.

## [2.2.1] — 2026-10-11

- Ícono unificado en los instaladores: el mismo diseño del favicon (vídeo/play
  con gradiente) en los paquetes de todas las plataformas (v2.2.0 todavía llevaba
  el ícono anterior por haberse compilado antes del merge del #34).

## [2.2.0] — 2026-10-11

Persistencia de preferencias, cola de conversiones por lote, estimación de
tamaño, códecs de vídeo con aceleración de GPU y ajustes por archivo: las
preferencias sobreviven al cierre, puedes comprimir varios vídeos de una vez, el
tamaño esperado aparece antes de comprimir, la tarjeta **Avanzado** cambia a
H.265 o a codificadores de hardware (NVENC/QSV/AMF/VideoToolbox) y cada vídeo
acepta escala, corte, remoción de audio, FPS, rotación y thumbnail.

### Agregado

- **Estimación de tamaño**: en modo tamaño, el panel "Tamaño objetivo" muestra un
  badge con el tamaño esperado de la salida — calculado con el mismo presupuesto
  de bits del encode 2-pass (ahora único, en `internal/estimate`) — y el ahorro
  frente al archivo original cuando el objetivo es menor. Nuevo binding
  `EstimateSize`; en modo CRF todavía no hay previsión y el badge no aparece.
- **Códec H.265/x265**: la tarjeta "Avanzado" del paso 02 permite elegir H.264 o
  H.265 por trabajo; el preset sigue con su códec por defecto. Junto con el 2-pass
  existente, el modo tamaño usa `libx265` con la misma precisión del x264 (ahora
  el builder de argumentos por codificador vive en `internal/encode`).
- **Aceleración de GPU (NVENC/QSV/AMF/VideoToolbox)**: el hardware es opcional por
  trabajo (opt-in), y los backends que el `ffmpeg` de este computador no tiene, o
  cuyo driver falló la sonda de 1 frame, aparecen deshabilitados — nuevo binding
  `GetEncoders` con caché y re-detección ("volver a comprobar"). Las GPUs del modo
  tamaño usan VBR 1-pass (destino aproximado, ±5–10%), y la UI avisa cuando un
  codificador de hardware está activo; el software sigue siendo el
  predeterminado.
- **Ajustes por archivo**: nueva tarjeta **"Ajustes por archivo"** en el paso 02
  con escala (predeterminado/original/personalizada), corte por ventana en
  `mm:ss`, remoción de audio, FPS, rotación (transpose) y generación de thumbnail
  al final del encode. Builder de filtros en `internal/compress/adjust.go` con
  orden fija transpose→scale→fps, `-ss/-to` antes de `-i`, presupuesto de tamaño
  y progreso basados en la duración cortada, validación de conflictos (corte
  manual × split, corte invertido, rotación/FPS fuera de rango) y nuevo binding
  `OpenThumbnailDialog`.
- **Preset NVENC (velocidad × calidad)**: con el codificador NVIDIA activo, la
  tarjeta **Avanzado** gana un selector de preset `p1`–`p7` por trabajo
  (predeterminado `p4`, equilibrado). `p1`/`p2` casi duplican la velocidad de
  encode en la engine de vídeo (que ya queda en 100%), con una leve pérdida de
  eficiencia de compresión en modo tamaño; `p7` prioriza la calidad. El encode de
  hardware no usa la engine gráfica — el "GPU %" del panel refleja el máximo
  entre gráficos y encoder.
- **Sidebar colapsable**: las tarjetas de configuración (`Tamaño objetivo`/
  `Calidad`, `Avanzado`, `Presets`, `Cortar en partes`, `Ajustes por archivo`)
  colapsan y expanden por su cabecera (chevron, clic o teclado). `Avanzado`,
  `Cortar en partes` y `Ajustes` vienen **cerrados** por defecto, reduciendo el
  ruido visual; el estado vale para la sesión.
- **Tiempo total de conversión**: cada trabajo de la cola muestra el tiempo
  transcurrido en tiempo real durante el proceso y el **total** al terminar (card
  de progreso, fila de la cola y card de resultado) — útil para comparar presets
  y codificadores (software × GPU, NVENC `p1`–`p7`).
- **Arrastrar y soltar en la ventana principal**: el contenido gana un área de
  destino (drop-target) con "arrastra un vídeo aquí" + botón **"abrir vídeo…"**,
  visible incluso con un vídeo ya cargado — soltar un archivo cambia la entrada y
  varios encolan (reuso del `wails:file-drop`).
- **Modo de depuración local**: `VIDCTL_DEBUG=1 ./build/bin/vidctl` enciende un
  log en la terminal con las transiciones de la cola, los eventos
  emitidos/recibidos (y los patches descartados por estado), las llamadas
  `Compress`/`GetTasks` y las excepciones de JS del frontend — para diagnosticar
  la UI de progreso sin afectar la ejecución normal (apagado por defecto).
- **Cola de conversiones (batch)**: "Agregar vídeos…" encola varios archivos de
  una vez (o suelta varios en la ventana), un trabajo por archivo con el preset y
  la salida actuales; la cola aparece en lugar del paso 04 con nombre, preset,
  estado, barra de progreso del elemento en ejecución, cancelación por elemento y
  "vaciar cola". Nuevos eventos `compress:queued`/`compress:start` y ruteo por
  `JobID` en el frontend: un error en un elemento no tumba a los demás. Nuevos
  bindings: `CompressMultiple`, `GetTasks` y `ClearFinished`.
- **Configuración persistente**: preset, tamaño en MB, CRF y carpeta de destino
  vuelven como estaban al siguiente inicio. Archivo en
  `<config-dir>/vidctl/config.json` (Linux `~/.config`, macOS
  `~/Library/Application Support`, Windows `%AppData%`), escrito de forma atómica
  para no corromperse en una caída. Un archivo ausente o inválido abre la app con
  los predeterminados en vez de fallar.
- **Preferencias → Binarios**: campos para apuntar el `ffmpeg` y el `ffprobe`
  fuera del PATH. La ruta configurada tiene prioridad sobre el PATH, vale al
  instante (sin reiniciar) y una ruta inválida se reporta como error en vez de
  caer silenciosamente al binario del PATH.
- **Preferencias → Salida**: campo de carpeta de destino; vacío mantiene guardar
  junto al vídeo original.

### Cambiado

- La barra lateral pasó a ser un **panel redondeado** (estilo GNOME/placa
  flotante): mismo color del contenido, con contorno suave (`18px` de radio) y
  sombra sutil en lugar de la columna "cuadrada" con línea completa de división —
  la separación ahora es por espacio y esquinas redondeadas, con sombra definida
  por tema (claro/oscuro) y sin depender del sistema.
- La **scrollbar de la barra lateral** es propia de la UI: la nativa
  (GTK/WebKit) se ocultó — no sale del panel ni crece sobre el contenido al pasar
  el mouse. La barra custom (fina, `8px`, recuada) aparece al desplazar/hover/
  arrastrar y se va en ~0,8s quieta, con el mismo comportamiento en las tres
  plataformas.
- El card de progreso sigue al elemento seleccionado en la cola y la cancelación
  ahora es por elemento — la acción ya no detiene todo el trabajo.
- `maxParallel` ahora vale: define cuántos vídeos se comprimen a la vez
  (predeterminado **1** = secuencial), aplicado al inicio y al guardar
  Preferencias.
- `CheckFFmpeg`, `GetMediaInfo` y la miniatura ahora respetan la ruta de binario
  configurada, con mensaje aparte para "no está en el PATH" y "la ruta que
  configuraste no funciona".
- Los campos `language`, `notifyOnDone` y `openFolderOnDone` ya se conservan en
  el archivo, pero solo tienen efecto cuando existan las funciones
  correspondientes (fases futuras del plan).
- El modo de depuración (`VIDCTL_DEBUG=1`) ahora rastrea los **comandos externos
  y la decisión del pipeline**: el comando `ffmpeg` completo (2-pass, single-pass
  y thumbnail), la elección de codificador/hardware por segmento
  (`encoder=hevc_nvenc`, 2-pass, cadena `-vf`, seek, remoción de audio, FPS,
  rotación), el `ffprobe` y la miniatura (`[media]`), y el listado/sonda de
  codificadores (`[encode]`, con el motivo cuando un backend falla). Cada comando
  es reproducible línea a línea.

### Corregido

- La detección de codificadores ya no traba la selección de archivo cuando un
  backend está listado por el `ffmpeg` pero falla la sonda de 1 frame (ej.:
  `h264_amf` sin GPU AMD): `codecs` venía `null` en el JSON y el frontend se caía
  al armar la lista de codificadores. Ahora la detección siempre devuelve una
  lista (vacía cuando no hay códecs) y el frontend también tolera `null`.
- El **"GPU %"** del panel de progreso medía solo `utilization.gpu`
  (gráficos/compute), que queda ~0 durante un encode NVENC — el encode de vídeo
  carga la engine de **encoder** (`utilization.encoder`), no la barra de gráficos.
  El colector ahora reporta el **máximo entre las dos**, así que el panel refleja
  el encode de hardware de verdad.
- La sonda de codificadores de hardware pasó a usar un frame `256×256`: el
  `64×64` anterior estaba **bajo el tamaño mínimo** del NVENC ("Frame Dimension
  less than the minimum supported value"), por lo que las GPUs NVIDIA modernas
  aparecían como "detectado pero no corrió" incluso con el driver funcionando.
- Backends por plataforma: **AMF (AMD)** solo se ofrece en Windows y
  **VideoToolbox** solo en macOS; NVENC y QSV siguen detectándose en cualquier SO
  y **Software**/"Automático" siempre están disponibles. Además, el motivo de la
  sonda fallida ahora aparece en la UI (tarjeta **Avanzado**) en vez de un texto
  genérico.
- **Arrastrar y soltar** funcionando en las tres plataformas: la app no solo
  escucha `wails:file-drop` sino que registra el `OnFileDrop` del runtime (que
  conecta los listeners de drop en el WebView). En **Linux** el handler nativo de
  Wails hace que WebKit navegue al archivo soltado (el vídeo "suena" y la UI
  desaparece), por lo que allí la ruta se lee del propio DOM (`text/uri-list`)
  sin navegación; Windows/macOS siguen con el drop nativo de Wails. Toda la
  ventana es el área de drop y, durante el arrastre, un **punteado alrededor de la
  pantalla** indica dónde soltar.
- **Salida y thumbnail por defecto junto a la fuente**: sin carpeta de destino
  guardada, las rutas sugeridas iban al directorio de trabajo de la app (el
  archivo del thumbnail "desaparecía" para quien buscaba junto al vídeo). Ahora el
  predeterminado es la carpeta del vídeo de origen (o la carpeta configurada en
  Preferencias).
- La **thumbnail** generada usa un frame **5 segundos después del inicio** del
  vídeo (primer segmento) en vez del medio del último tramo — en las divisiones
  en partes caía cerca del final y parecía el "último frame"; además evita
  pantallas negras de apertura. En vídeos cortos (menos de 10s) cae a la mitad.

## [2.1.0] — 2026-09-27

Rediseño visual en el patrón Constrict con arrastrar y soltar, miniaturas y
ajuste fino de los contadores.

### Agregado

- **Visual en el patrón Constrict**: tema oscuro mate (tokens recientes con accent
  slate), tema claro derivado e ícono propio en los instaladores de
  Linux/Windows/macOS; layout de dos columnas (sidebar de ajustes + área del
  vídeo), cards por grupo, y el modal de Preferencias (tema sistema/claro/oscuro
  y recheck del ffmpeg).
- **Arrastrar y soltar**: suelta un vídeo en la ventana para seleccionarlo; el
  área vacía queda resaltada en hover (`--wails-drop-target`, evento
  `wails:file-drop`).
- **Miniaturas**: la fila del vídeo muestra un frame extraído vía ffmpeg
  (buscando en `min(1s, dur/2)`, limitado a 320px), cacheado por path+size+mtime
  en `os.TempDir()/vidctl-thumbs`; nuevo binding `GetThumbnail`.
- **Contadores con valor editable y ajuste fino**: clic en el número para
  escribir (Enter/clic fuera confirma, Esc cancela, coma aceptada) y manten **Shift**
  en las flechas `+/−` (o `↑/↓`) para un paso fino de 0,1.
- **min/parte fraccionario de punta a punta**: `split.Spec.MinutesEach` acepta
  fracción (`1.5` = 1 m 30 s) del frontend al corte, con la capa de
  `internal/split` testeada.
- **Menú de la fila del vídeo (⋮)**: cambiar archivo / limpiar.
- Rediseño del ícono de la app: fondo oscuro redondeado con glyph de compresión
  en accent slate (appicon.png, icon.ico y el pkg/icon de Linux).

### Cambiado

- Fase 4 del plan Constrict entregada: componentes base (Stepper, Toggle,
  RadioCardGroup, Modal, InfoTip, KebabMenu) sin estilos scoped, movidos al
  design system global.
- `internal/media` ahora también genera miniaturas; documentación de
  Preferencias, pipeline y estructura actualizada en el README.

## [2.0.1] — 2026-09-22

Retoca la experiencia de la v2.0.0: calidad asistida y ajustes de interfaz.

### Agregado

- **Aviso de calidad pre-encode**: estimación del bitrate de vídeo (por parte
  cuando hay corte) comparada con un mínimo por franja de resolución; cuando es
  baja, la sidebar muestra el motivo y una sugerencia aplicable con un clic —
  subir el objetivo en MB, cortar en partes de X min o migrar a un preset CRF.
- Card de **resumen del trabajo** en el espacio libre bajo el vídeo cuando está
  ocioso: destino + objetivo, plan de corte y bitrate estimado vs. mínimo — el
  mismo lugar que el card de progreso ocupa durante el proceso.
- Landing page de la era v2 en GitHub Pages, con demo animada fiel a la UI real
  (elección de preset → corte → procesamiento → tema claro).

### Cambiado

- Sidebar: el grupo de elección de preset pasó de "Destino" a **"Presets"**.
- Sidebar más ancha (300→344px) para acomodar los banners de corte/calidad.
- Release pasó a ser 100% intencional: el push en `main` ya no dispara el
  workflow (solo `gh workflow run release.yml --ref vX.Y.Z` en una tag).

### Corregido

- Las descripciones de los presets en la sidebar ahora hacen salto de línea y
  muestran el texto completo (antes se veían cortadas con "…" en Instagram/
  YouTube).

- Las descripciones de los presets en la sidebar ahora hacen salto de línea y
  muestran el texto completo (antes se veían cortadas con "…" en Instagram/
  YouTube).
- Sidebar más ancha (300→344px) y card de **resumen del trabajo** en el espacio
  libre bajo el vídeo cuando está ocioso: destino + objetivo, plan de corte y
  bitrate estimado vs. mínimo — el mismo lugar que el card de progreso ocupa
  durante el proceso.
- **Aviso de calidad pre-encode**: estimación del bitrate de vídeo (por parte
  cuando hay corte) comparada con un mínimo por franja de resolución; cuando es
  baja, la sidebar muestra el motivo y una sugerencia aplicable con un clic —
  subir el objetivo en MB, cortar en partes de X min o migrar a un preset CRF.
- Sidebar: el grupo de elección de preset pasó de "Destino" a **"Presets"**.
- Release pasó a ser 100% intencional: el push en `main` ya no dispara el
  workflow (solo `gh workflow run release.yml --ref vX.Y.Z` en una tag).

## [2.0.0] — 2026-09-22

Primera gran revisión de la interfaz + corte de vídeo por tiempo.

### Agregado

- **Corte en partes (split)**: divide el vídeo por tiempo — N partes iguales o X
  minutos por parte (ej.: 30 min → 6 × 5 min) — con mínimo de 1 min por parte y
  tope de 60 partes; cada parte pasa por la compresión elegida y sale como
  `nombre-partN.mp4` (`internal/split` + `Job.Split` en el pipeline).
- **Rediseño de la interfaz en estilo Constrict** (branch `feat/ui-constrict`):
  layout full-height con barra lateral de controles (Destino, Calidad/Tamaño
  objetivo, Cortar en partes, Salida), estado vacío estilo StatusPage, fila del
  vídeo como card con miniatura/estado/PIE de progreso y barra de acciones
  inferior.
- **Tema claro/oscuro manual**: botón sol/luna arriba; sigue el sistema por
  defecto y la elección se conserva entre ejecuciones.
- **Panel de progreso enriquecido**: card propio bajo el vídeo (mismo ancho y
  radio) con porcentaje agregado por parte y lectura en vivo de CPU, memoria, %
  del ffmpeg y GPU (NVIDIA cuando está disponible) — nuevo `internal/sysinfo` vía
  polling de 1s (`GetUsage`).
- `OpenMultipleDialog` en el backend para selección múltiple de archivos (base
  de la futura cola batch).
- Roadmap: fase 11 (corte en segmentos) y plan de UI
  `docs/plano-ui-constrict.md`.

### Corregido

- Pie/acción con franja vacía gigante en ventanas altas (el layout ahora ancla
  `html/body/#app` al 100% con columna flex).
- Scroll horizontal no deseado en la sidebar (min-width de flex children +
  overflow-x).
- Ícono del estado vacío (cambio de "pila" a cámara de vídeo).

## [1.0.17] — 2026-09-13

Primera versión **production ready** de vidctl.

### Agregado

- Compresión de vídeo con presets por plataforma: WhatsApp Status (10 MB),
  WhatsApp (16 MB), Instagram (16 MB), Shorts (30 MB), YouTube (CRF) o un tamaño
  libre definido por el usuario (2–100 MB).
- Encode ffmpeg **2-pass** para entregar el tamaño máximo sin pasar el límite
  elegido.
- Progreso en tiempo real por etapa (pass 1/2 y pass 2/2) con cancelación en
  cualquier momento.
- Verificación de `ffmpeg`/`ffprobe` al inicio, con el botón "volver a comprobar"
  que relee el PATH del sistema — permite instalar el ffmpeg con la app abierta
  sin reiniciar.
- Escritorio nativo para **Linux, Windows y macOS** (Wails v2 + WebKit/WebView2).
- Modo offline: fuentes e interfaz embebidas en el binario.
- Paquetes por plataforma (.zip, .deb y .rpm) con `checksums.txt` publicado y
  verificado en todas las releases.
- Escala predeterminada con ancho máximo de 1280 px conservando el aspecto del
  vídeo.

### Corregido

- Windows: las conversiones y las sondas de archivo ya no abren ventanas de
  consola (`CREATE_NO_WINDOW`).
- Windows: "volver a comprobar" detecta un ffmpeg/ffprobe recién instalado sin
  reiniciar la app (relectura del PATH de usuario/máquina en el registro).
- Los paquetes .zip de release ya no incluían todo el árbol de recursos del
  proyecto.

### Especificaciones

- Go 1.25+, Wails v2.15, Svelte 5, ffmpeg/ffprobe requeridos en el sistema.
- Cada release publica solo los paquetes de su sistema operativo (Windows:
  `.zip`; Linux: `.zip`, `.deb`, `.rpm`; macOS: `.zip` con `vidctl.app`).

[1.0.17]: https://github.com/fabianoflorentino/vidctl/releases/tag/v1.0.17