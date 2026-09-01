# BoraIF

O BoraIF é um sistema web para escolas de Ensino Médio criarem,
organizarem e reaproveitarem questões de prova de múltipla escolha — e, a
partir delas, montarem cadernos de prova prontos em PDF.

## Em poucas palavras

Um professor entra no sistema, escreve uma questão num editor de texto
completo (com formatação, imagens e fórmulas matemáticas), pede uma
revisão crítica para uma inteligência artificial se quiser, e o sistema
salva tudo sozinho enquanto ele digita. Mais tarde, alguém responsável
pela aplicação da prova junta questões de várias disciplinas, define
quantas de cada uma entram, e gera um PDF pronto para imprimir — em
segundo plano, sem travar a tela de ninguém.

O sistema cobre as 13 disciplinas do Ensino Médio brasileiro: Língua
Portuguesa, Língua Inglesa, Física, Química, Redação, História,
Geografia, Matemática, Biologia, Artes, Educação Física, Filosofia e
Sociologia.

## O que dá para fazer

- **Escrever questões** com um editor rico de verdade: negrito, itálico,
  listas, tabelas, imagens e fórmulas matemáticas — tanto para quem sabe
  LaTeX quanto para quem prefere montar a fórmula visualmente.
- **Nunca perder o que foi escrito**: o sistema salva automaticamente
  enquanto o professor digita, sem travar nem sobrecarregar nada.
- **Reaproveitar imagens** já enviadas por outros professores da mesma
  disciplina.
- **Pedir uma segunda opinião a uma IA** sobre a clareza, a coerência e a
  qualidade de uma questão — a IA só analisa e sugere, quem decide o que
  muda é sempre o professor.
- **Organizar provas por temporada** (ex.: "2026/1"), cada uma podendo ter
  mais de um caderno diferente, cada caderno com sua própria mistura de
  disciplinas, assuntos e dificuldades.
- **Gerar o PDF final** pronto para impressão em papel A4, com numeração
  de página e cabeçalho — o sistema avisa antes se não houver questões
  suficientes para o que foi pedido.

## Quem usa o quê

- **Professores** escrevem e revisam questões da própria disciplina.
- **Administradores** cuidam de usuários, assuntos e das configurações
  gerais do sistema.
- **Gestores de aplicação** organizam as campanhas de prova e mandam
  gerar os PDFs.

## Tecnologia usada (resumo)

Sem entrar em detalhes técnicos (isso fica em `architecture.md`): o
sistema é dividido em três partes que rodam juntas — o que a pessoa vê no
navegador é construído em **JavaScript/TypeScript** com **React**; a parte
que guarda e processa as informações é escrita em **Go**; e os dados ficam
num banco **PostgreSQL**. O editor de texto usa uma biblioteca
especializada (TipTap) e as fórmulas matemáticas são renderizadas com
KaTeX. A geração de PDF usa um navegador Chromium rodando "escondido" nos
bastidores. Tudo isso roda dentro de três containers Docker, o que torna
instalar e atualizar o sistema bem mais simples.

## Quero rodar o sistema

- **Caminho recomendado (com Docker)**: veja `RUNNINGDOCKER.MD` (guia
  completo) ou `running.md` (resumo).
- **Sem Docker**: veja `withoutdocker.md`.

## Quero entender melhor o projeto

- `project.md` — o que foi construído, fase a fase.
- `architecture.md` — como o sistema é organizado por dentro.
- `database.md` — o banco de dados, tabela por tabela.
- `DEVELOPMENT.md` — histórico técnico completo: decisões, endpoints,
  bugs encontrados e corrigidos ao longo do desenvolvimento.
- `especificacoes.md` — o documento de requisitos original que guiou a
  construção deste sistema do começo ao fim.
