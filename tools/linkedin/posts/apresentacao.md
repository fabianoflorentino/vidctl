Cansado de bater a cabeça para mandar o vídeo do tamanho certo?

WhatsApp, Instagram, YouTube... cada plataforma tem um limite diferente — e na
hora de "encurtar" a gente apela pra cortar cena e perder qualidade.

Apresento o vidctl: um app desktop (Linux, Windows e macOS) que resolve isso de
vez. Você escolhe o destino e ele calcula o bitrate sozinho, com encode 2-pass
do ffmpeg — mantendo duração, formato e qualidade. 100% offline.

No teste com um vídeo real: 1m33s · Full HD vertical · 22,1 MB → 9,7 MB, dentro
do limite de 10 MB do WhatsApp, sem cortar um segundo.

Feito com Go + Wails + Svelte, open source.

Se você envia vídeo todo dia (cliente, redes sociais, escola), vale dois minutos
pra conhecer.

#opensource #ffmpeg #video #desenvolvimento #golang #produtividade