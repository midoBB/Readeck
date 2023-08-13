// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Application} from "@hotwired/stimulus"
import {definitions} from "stimulus:./controllers"

const application = Application.start()
application.load(definitions)

import "./lib/turbo"
