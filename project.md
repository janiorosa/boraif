# O que é este projeto

## BoraIF

O BoraIF é um sistema para escolas elaborarem, organizarem, revisarem e
reutilizarem questões de múltipla escolha do Ensino Médio brasileiro, e
depois montarem cadernos de prova em PDF a partir delas. Ele foi
construído inteiramente a partir de um documento de especificação
(`especificacoes.md`), seguindo-o passo a passo, fase por fase.

Disciplinas suportadas: Língua Portuguesa, Língua Inglesa, Física,
Química, Redação, História, Geografia, Matemática, Biologia, Artes,
Educação Física, Filosofia e Sociologia.

## Para quem é

- **Professores (papel ELABORADOR)**: cada um associado a uma disciplina,
  escrevem e revisam questões da própria disciplina, usam o assistente de
  revisão por IA e reutilizam imagens já enviadas por colegas.
- **Administradores (papel ADMIN)**: gerenciam usuários, disciplinas
  (fixas), assuntos e a configuração padrão de provas; têm acesso a tudo.
- **Gestores de aplicação (papel GESTOR)**: criam e configuram as
  campanhas de prova (aplicações) e seus cadernos, e disparam a geração
  dos PDFs.

## O que foi construído

O projeto foi implementado em 11 fases, na ordem sugerida pela própria
especificação — cada fase termina com o sistema inteiro compilando e
funcionando antes de avançar para a próxima. **As 11 fases estão
concluídas.**

### 1 — Fundação

Estrutura do repositório, Docker Compose com os 3 containers (frontend,
backend, PostgreSQL), banco de dados criado por migrations versionadas e
aplicadas automaticamente, esqueleto do backend em Go e do frontend em
React, e login por sessão.

### 2 — Usuários e permissões

CRUD de usuários (só ADMIN), três papéis (ADMIN/ELABORADOR/GESTOR),
autorização validada sempre no backend — nunca só escondendo botão na
tela.

### 3 — Assuntos

Cada assunto pertence a exatamente uma disciplina. Um professor pode criar
assunto novo para a própria disciplina; o sistema avisa (e pede
confirmação) se o nome for igual ou muito parecido a um já existente, para
evitar duplicação acidental.

### 4 — CRUD estrutural de questões

Uma questão tem enunciado, comando e cinco alternativas como entidades
separadas (nunca um bloco único de texto). A regra mais importante do
sistema — **exatamente cinco alternativas, exatamente uma correta** — é
garantida em duas camadas: validação no backend antes de qualquer
gravação, e uma trava no próprio banco de dados como segunda linha de
defesa.

### 5 — Editor de questões (TipTap)

Editor de texto rico completo para cada um dos sete campos de uma questão
(enunciado, comando, alternativas A a E): negrito, itálico, sublinhado,
tachado, sub/sobrescrito, listas, alinhamento, links, tabelas, imagens e
fórmulas matemáticas — inseridas tanto visualmente (sem precisar saber
LaTeX) quanto digitando LaTeX direto, com pré-visualização ao vivo.

### 6 — Salvamento automático (autosave)

A questão inteira é salva sozinha, com uma pequena espera depois da
última tecla digitada (nunca a cada caractere), mostrando na tela se está
salvando, se já salvou, ou se deu erro. Um botão de salvar manual continua
disponível.

### 7 — Biblioteca de imagens

Imagens enviadas por um professor ficam disponíveis para todos os
professores da mesma disciplina — sem permissões complicadas por imagem
individual. Dá para buscar, reutilizar e inserir direto no editor, ou
enviar uma nova sem sair da questão que está sendo escrita.

### 8 — Assistente de revisão por Inteligência Artificial

A IA **não escreve questões**: ela só analisa o que o professor já
escreveu (enunciado, comando, alternativas, ou a questão inteira) e devolve
uma crítica — clareza, ambiguidade, se os distratores são plausíveis, se
existe mais de uma resposta certa, adequação ao ano escolar, e por aí vai.
O professor decide o que aceitar; nada é alterado sozinho. Cada professor
usa a própria chave de acesso à OpenAI, cadastrada uma vez e guardada
cifrada — nunca em texto puro, e nunca reexibida depois de salva.

### 9 — Aplicações e cadernos de prova

Uma aplicação é uma campanha de prova (ex.: "2026/1"). Cada aplicação pode
ter mais de um **caderno** — o mais comum são dois, mas pode ser qualquer
quantidade — e cada caderno tem sua própria configuração: quantas
questões, de quais anos, quantas de cada disciplina (podendo refinar por
assunto e/ou dificuldade). Existe uma configuração padrão que já vem
pré-preenchida em todo caderno novo, editável depois sem afetar o padrão
nem os outros cadernos. Antes de qualquer geração, o sistema mostra
exatamente quantas questões foram pedidas versus quantas existem de fato
disponíveis, cota por cota — nunca deixa começar uma geração fadada a
falhar.

### 10 — Geração de PDF

Ao pedir a geração de um caderno, o sistema sorteia as questões elegíveis
de cada cota, congela essa seleção (o PDF de uma prova já feita nunca muda,
mesmo que a questão original seja editada depois), e gera o arquivo em
segundo plano — sem travar a tela nem a navegação enquanto isso acontece.
O PDF sai em A4, com cabeçalho, rodapé, numeração de página, imagens e
fórmulas matemáticas renderizadas de verdade (não como texto cru).

### 11 — Refinamento

Testes automatizados cobrindo as regras de negócio mais críticas (a regra
das 5 alternativas/1 correta, permissões por disciplina, cifra da API Key,
conversão do editor para HTML do PDF), revisão de segurança contínua a
cada fase (nunca só no final), e a documentação que você está lendo agora.

## O estado atual, em uma frase

Um professor consegue entrar no sistema, escrever uma questão completa com
formatação e fórmulas, pedir uma revisão por IA, salvar tudo
automaticamente, e essa questão fica disponível para, mais tarde, um
gestor montar um caderno de prova e gerar o PDF final — tudo dentro de um
sistema simples de rodar (três containers Docker) e barato de manter.

## Documentos relacionados

- `README.md` — visão geral rápida, para qualquer pessoa.
- `architecture.md` — como o sistema é organizado por dentro, tecnologia
  usada, pastas importantes.
- `database.md` — cada tabela do banco de dados, explicada.
- `RUNNINGDOCKER.MD` / `running.md` — como colocar para rodar com Docker.
- `withoutdocker.md` — como colocar para rodar sem Docker.
- `DEVELOPMENT.md` — histórico técnico completo, fase a fase, com todas as
  decisões de arquitetura e os bugs encontrados e corrigidos nas revisões.
- `especificacoes.md` — o documento de requisitos original, que guiou
  cada decisão deste projeto.
