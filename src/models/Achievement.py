from typing import Callable, Dict


class Achievement:
    registry: Dict[str, 'Achievement'] = {}

    def __init__(self, ach_id: str, name: str, emoji: str, glory: int, desc: str, check_func: Callable[[dict], bool]):
        self.id = ach_id
        self.name = name
        self.emoji = emoji
        self.glory = glory
        self.desc = desc

        self.check_func = check_func

        Achievement.registry[self.id] = self

    def is_unlocked(self, player_stats: dict) -> bool:
        return self.check_func(player_stats)

    @classmethod
    def get(cls, ach_id: str) -> 'Achievement | None':
        return cls.registry.get(ach_id)

    @classmethod
    def get_all(cls) -> list['Achievement']:
        return list(cls.registry.values())


Achievement("pvp_rookie", "Combattant en herbe", "⚔️", 10, "Remporte ton premier combat PvP.",
            lambda s: s.get("pvp_wins", 0) >= 1)

Achievement("pvp_gladiator", "Gladiateur", "🏟️", 50, "Remporte 50 combats PvP.",
            lambda s: s.get("pvp_wins", 0) >= 50)

Achievement("pvp_punching_bag", "Sac de frappe", "🩹", 5, "Perds 10 combats PvP. Courage...",
            lambda s: s.get("pvp_losses", 0) >= 10)

# --- COMBAT PVE (Chasse) ---
Achievement("pve_hunter", "Chasseur de monstres", "🌲", 20, "Gagne 25 combats en expédition (PvE).",
            lambda s: s.get("pve_wins", 0) >= 25)

# --- ÉCONOMIE & MÉTIERS ---
Achievement("eco_1k", "Nouveau Riche", "💵", 20, "Atteins 1 000 $ sur ton compte.",
            lambda s: s.get("balance", 0) >= 1000)

Achievement("eco_10k", "Rentier", "💸", 50, "Atteins 10 000 $ sur ton compte.",
            lambda s: s.get("balance", 0) >= 10000)

Achievement("eco_50k", "Loup de Wall Street", "📈", 100, "Atteins 50 000 $ sur ton compte.",
            lambda s: s.get("balance", 0) >= 50000)

Achievement("eco_100k", "Millionnaire en Devenir", "🤑", 200, "Atteins 100 000 $ sur ton compte.",
            lambda s: s.get("balance", 0) >= 100000)

Achievement("eco_1m", "Billionaire", "👑", 500, "Atteins 1 000 000 $ sur ton compte.",
            lambda s: s.get("balance", 0) >= 1000000)

Achievement("eco_rich", "Capitaliste", "💰", 100, "Gagne un total cumulé de 10 000 $.",
            lambda s: s.get("money_earned", 0) >= 10000)

Achievement("job_miner", "Nain de Moria", "⛏️", 30, "Mine 100 minerais.",
            lambda s: s.get("items_mined", 0) >= 100)

Achievement("pet_feeder", "Chef Étoilé", "🍖", 20, "Donne 50 objets à manger à tes familiers.",
            lambda s: s.get("pets_fed", 0) >= 50)

Achievement("pet_level_10", "Éleveur Novice", "🥚", 20, "Fais monter un familier au niveau 2.",
            lambda s: s.get("max_pet_level", 0) >= 2)

Achievement("pet_level_20", "Dresseur", "🐾", 50, "Fais monter un familier au niveau 5.",
            lambda s: s.get("max_pet_level", 0) >= 5)

Achievement("pet_level_50", "Maître des Monstres", "🐉", 100, "Fais monter un familier au niveau 10.",
            lambda s: s.get("max_pet_level", 0) >= 10)

Achievement("pet_level_100", "Légende Vivante", "✨", 300, "Fais monter un familier au niveau 20.",
            lambda s: s.get("max_pet_level", 0) >= 20)

# --- CASINO ---
# Coinflip - Won
Achievement("coinflip_won_1k", "Chance du débutant", "🪙", 10, "Gagne 1 000 $ au Coinflip.",
            lambda s: s.get("coinflip_money_won", 0) >= 1000)
Achievement("coinflip_won_5k", "Habitué de la pièce", "🪙", 20, "Gagne 5 000 $ au Coinflip.",
            lambda s: s.get("coinflip_money_won", 0) >= 5000)
Achievement("coinflip_won_100k", "Double ou Rien", "🪙", 100, "Gagne 100 000 $ au Coinflip.",
            lambda s: s.get("coinflip_money_won", 0) >= 100000)
Achievement("coinflip_won_1m", "Maître Pile ou Face", "💰", 500, "Gagne 1 000 000 $ au Coinflip.",
            lambda s: s.get("coinflip_money_won", 0) >= 1000000)

# Coinflip - Lost
Achievement("coinflip_lost_1k", "Mauvaise passe", "🌧️", 10, "Perds 1 000 $ au Coinflip.",
            lambda s: s.get("coinflip_money_lost", 0) >= 1000)
Achievement("coinflip_lost_5k", "Poissard", "🌧️", 20, "Perds 5 000 $ au Coinflip.",
            lambda s: s.get("coinflip_money_lost", 0) >= 5000)
Achievement("coinflip_lost_100k", "Ruiné par une pièce", "💸", 100, "Perds 100 000 $ au Coinflip.",
            lambda s: s.get("coinflip_money_lost", 0) >= 100000)
Achievement("coinflip_lost_1m", "Le Dindon de la Farce", "🤡", 500, "Perds 1 000 000 $ au Coinflip.",
            lambda s: s.get("coinflip_money_lost", 0) >= 1000000)

# Slots - Won
Achievement("slots_won_1k", "Tireur chanceux", "🎰", 10, "Gagne 1 000 $ aux Slots.",
            lambda s: s.get("slots_money_won", 0) >= 1000)
Achievement("slots_won_5k", "Poche pleine de jetons", "🎰", 20, "Gagne 5 000 $ aux Slots.",
            lambda s: s.get("slots_money_won", 0) >= 5000)
Achievement("slots_won_100k", "L'Alignement Parfait", "🎰", 100, "Gagne 100 000 $ aux Slots.",
            lambda s: s.get("slots_money_won", 0) >= 100000)
Achievement("slots_won_1m", "Jackpot Ambulant", "🎰", 500, "Gagne 1 000 000 $ aux Slots.",
            lambda s: s.get("slots_money_won", 0) >= 1000000)

# Slots - Lost
Achievement("slots_lost_1k", "La machine est truquée", "😠", 10, "Perds 1 000 $ aux Slots.",
            lambda s: s.get("slots_money_lost", 0) >= 1000)
Achievement("slots_lost_5k", "Bandit Manchot", "😠", 20, "Perds 5 000 $ aux Slots.",
            lambda s: s.get("slots_money_lost", 0) >= 5000)
Achievement("slots_lost_100k", "Accro au levier", "📉", 100, "Perds 100 000 $ aux Slots.",
            lambda s: s.get("slots_money_lost", 0) >= 100000)
Achievement("slots_lost_1m", "Gros Donateur du Casino", "💸", 500, "Perds 1 000 000 $ aux Slots.",
            lambda s: s.get("slots_money_lost", 0) >= 1000000)

# Blackjack - Won
Achievement("blackjack_won_1k", "21 ! (1k$)", "🃏", 10, "Gagne 1 000 $ au Blackjack.",
            lambda s: s.get("blackjack_money_won", 0) >= 1000)
Achievement("blackjack_won_5k", "Compteur de cartes (5k$)", "🃏", 20, "Gagne 5 000 $ au Blackjack.",
            lambda s: s.get("blackjack_money_won", 0) >= 5000)
Achievement("blackjack_won_100k", "Le Croupier Pleure (100k$)", "🃏", 100, "Gagne 100 000 $ au Blackjack.",
            lambda s: s.get("blackjack_money_won", 0) >= 100000)
Achievement("blackjack_won_1m", "As de Pique des As (1M$)", "🃏", 500, "Gagne 1 000 000 $ au Blackjack.",
            lambda s: s.get("blackjack_money_won", 0) >= 1000000)

# Blackjack - Lost
Achievement("blackjack_lost_1k", "Trop Gourmand (1k$)", "🤦", 10, "Perds 1 000 $ au Blackjack.",
            lambda s: s.get("blackjack_money_lost", 0) >= 1000)
Achievement("blackjack_lost_5k", "Bust Constant (5k$)", "🤦", 20, "Perds 5 000 $ au Blackjack.",
            lambda s: s.get("blackjack_money_lost", 0) >= 5000)
Achievement("blackjack_lost_100k", "Dépression sur table (100k$)", "😔", 100, "Perds 100 000 $ au Blackjack.",
            lambda s: s.get("blackjack_money_lost", 0) >= 100000)
Achievement("blackjack_lost_1m", "Tu as donné ta maison (1M$)", "🏚️", 500, "Perds 1 000 000 $ au Blackjack.",
            lambda s: s.get("blackjack_money_lost", 0) >= 1000000)

# Roulette - Won
Achievement("roulette_won_1k", "Miraculé (1k$)", "🔫", 10, "Gagne 1 000 $ à la Roulette.",
            lambda s: s.get("roulette_money_won", 0) >= 1000)
Achievement("roulette_won_5k", "Trompe la Mort (5k$)", "🔫", 20, "Gagne 5 000 $ à la Roulette.",
            lambda s: s.get("roulette_money_won", 0) >= 5000)
Achievement("roulette_won_100k", "Immortalité (100k$)", "🔫", 100, "Gagne 100 000 $ à la Roulette.",
            lambda s: s.get("roulette_money_won", 0) >= 100000)
Achievement("roulette_won_1m", "Survivant Légendaire (1M$)", "🔫", 500, "Gagne 1 000 000 $ à la Roulette.",
            lambda s: s.get("roulette_money_won", 0) >= 1000000)

# Roulette - Lost
Achievement("roulette_lost_1k", "Première blessure (1k$)", "🩸", 10, "Perds 1 000 $ à la Roulette.",
            lambda s: s.get("roulette_money_lost", 0) >= 1000)
Achievement("roulette_lost_5k", "Abonné à l'hôpital (5k$)", "🩸", 20, "Perds 5 000 $ à la Roulette.",
            lambda s: s.get("roulette_money_lost", 0) >= 5000)
Achievement("roulette_lost_100k", "Suicidaire (100k$)", "💀", 100, "Perds 100 000 $ à la Roulette.",
            lambda s: s.get("roulette_money_lost", 0) >= 100000)
Achievement("roulette_lost_1m", "Cimetière Personnel (1M$)", "🪦", 500, "Perds 1 000 000 $ à la Roulette.",
            lambda s: s.get("roulette_money_lost", 0) >= 1000000)

# --- BETTING ---
Achievement("bet_rookie", "Parieur du Dimanche", "🎲", 10, "Gagne un pari.",
            lambda s: s.get("wagers_won", 0) >= 1)

Achievement("bet_pro", "Visionnaire", "🔮", 50, "Gagne 25 paris.",
            lambda s: s.get("wagers_won", 0) >= 25)



# --- LOTTO ---
Achievement("lotto_rookie", "Joueur Régulier", "🎫", 10, "Joue 10 tickets de Loto.",
            lambda s: s.get("lotto_participations", 0) >= 10)

Achievement("lotto_winner", "Le Grand Gagnant", "🎉", 200, "Remporte le Jackpot du Loto !",
            lambda s: s.get("lotto_won", 0) >= 1)

# --- ECONOMY / DAILY ---
Achievement("daily_1", "Travailleur Engagé", "📅", 10, "Réclame ta paye journalière pour la 1ère fois.",
            lambda s: s.get("daily_uses", 0) >= 1)

Achievement("daily_10", "Employé du Mois", "📅", 20, "Réclame ta paye journalière 10 fois.",
            lambda s: s.get("daily_uses", 0) >= 10)
            
Achievement("daily_50", "Routine Matinale", "📅", 50, "Réclame ta paye journalière 50 fois.",
            lambda s: s.get("daily_uses", 0) >= 50)
            
Achievement("daily_100", "L'Habitude fait le Moine", "📅", 100, "Réclame ta paye journalière 100 fois.",
            lambda s: s.get("daily_uses", 0) >= 100)

Achievement("daily_365", "Vétéran d'un An", "📅", 500, "Réclame ta paye journalière 365 fois (Soit 1 an complet !).",
            lambda s: s.get("daily_uses", 0) >= 365)


# --- PET COLLECTION ---
Achievement("pet_collector_common", "Zoologiste Amateur", "🦋", 50, "Possède tous les familiers Communs (8).",
            lambda s: s.get("collected_common_pets", 0) >= 8)

Achievement("pet_collector_rare", "Zoologiste Confirmé", "🦁", 150, "Possède tous les familiers Rares (6).",
            lambda s: s.get("collected_rare_pets", 0) >= 6)

Achievement("pet_collector_epic", "Expert Animalier", "🦄", 300, "Possède tous les familiers Épiques (4).",
            lambda s: s.get("collected_epic_pets", 0) >= 4)

Achievement("pet_collector_legendary", "Dresseur de Légendes", "🐉", 500, "Possède un familier Légendaire.",
            lambda s: s.get("collected_legendary_pets", 0) >= 1)

Achievement("pet_collector_all", "Maître du Monde Animal", "🌍", 1000, "Possède absolument toutes les espèces de familiers (19).",
            lambda s: s.get("collected_all_pets", 0) >= 19)

# --- PET RANKS ---
Achievement("rank_bronze", "Compétiteur Né", "🥉", 50, "Place un familier au rang Bronze.",
            lambda s: any("Bronze" in r for r in s.get("pet_ranks", [])))

Achievement("rank_silver", "Challenger", "🥈", 100, "Place un familier au rang Argent.",
            lambda s: any("Argent" in r for r in s.get("pet_ranks", [])))

Achievement("rank_gold", "Gladiateur Doré", "🥇", 500, "Place un familier au rang Or.",
            lambda s: any("Or" in r for r in s.get("pet_ranks", [])))

Achievement("rank_diamond", "L'Élite", "💎", 1000, "Place un familier au rang Diamant.",
            lambda s: any("Diamant" in r for r in s.get("pet_ranks", [])))

Achievement("rank_top5", "Maître de l'Olympe", "🌟", 5000, "Hisse un familier dans le Top 5 Mondial.",
            lambda s: any("Top 5" in r for r in s.get("pet_ranks", [])))