from typing import Callable, Dict
from src.utils.i18n import t


class Achievement:
    registry: Dict[str, 'Achievement'] = {}

    def __init__(self, ach_id: str, emoji: str, glory: int, check_func: Callable[[dict], bool]):
        self.id = ach_id
        self.emoji = emoji
        self.glory = glory
        self.check_func = check_func

        Achievement.registry[self.id] = self

    def name(self, lang: str) -> str:
        return t(f"achievements.{self.id}.name", lang)

    def desc(self, lang: str) -> str:
        return t(f"achievements.{self.id}.desc", lang)

    def is_unlocked(self, player_stats: dict) -> bool:
        return self.check_func(player_stats)

    @classmethod
    def get(cls, ach_id: str) -> 'Achievement | None':
        return cls.registry.get(ach_id)

    @classmethod
    def get_all(cls) -> list['Achievement']:
        return list(cls.registry.values())


Achievement("pvp_rookie", "⚔️", 10,
            lambda s: s.get("pvp_wins", 0) >= 1)

Achievement("pvp_gladiator", "🏟️", 50,
            lambda s: s.get("pvp_wins", 0) >= 50)

Achievement("pvp_punching_bag", "🩹", 5,
            lambda s: s.get("pvp_losses", 0) >= 10)

# --- COMBAT PVE (Chasse) ---
Achievement("pve_hunter", "🌲", 20,
            lambda s: s.get("pve_wins", 0) >= 25)

# --- ÉCONOMIE & MÉTIERS ---
Achievement("eco_1k", "💵", 20,
            lambda s: s.get("balance", 0) >= 1000)

Achievement("eco_10k", "💸", 50,
            lambda s: s.get("balance", 0) >= 10000)

Achievement("eco_50k", "📈", 100,
            lambda s: s.get("balance", 0) >= 50000)

Achievement("eco_100k", "🤑", 200,
            lambda s: s.get("balance", 0) >= 100000)

Achievement("eco_1m", "👑", 500,
            lambda s: s.get("balance", 0) >= 1000000)

Achievement("eco_rich", "💰", 100,
            lambda s: s.get("money_earned", 0) >= 10000)

Achievement("job_miner", "⛏️", 30,
            lambda s: s.get("items_mined", 0) >= 100)

Achievement("pet_feeder", "🍖", 20,
            lambda s: s.get("pets_fed", 0) >= 50)

Achievement("pet_level_10", "🥚", 20,
            lambda s: s.get("max_pet_level", 0) >= 2)

Achievement("pet_level_20", "🐾", 50,
            lambda s: s.get("max_pet_level", 0) >= 5)

Achievement("pet_level_50", "🐉", 100,
            lambda s: s.get("max_pet_level", 0) >= 10)

Achievement("pet_level_100", "✨", 300,
            lambda s: s.get("max_pet_level", 0) >= 20)

# --- CASINO ---
# Coinflip - Won
Achievement("coinflip_won_1k", "🪙", 10,
            lambda s: s.get("coinflip_money_won", 0) >= 1000)
Achievement("coinflip_won_5k", "🪙", 20,
            lambda s: s.get("coinflip_money_won", 0) >= 5000)
Achievement("coinflip_won_100k", "🪙", 100,
            lambda s: s.get("coinflip_money_won", 0) >= 100000)
Achievement("coinflip_won_1m", "💰", 500,
            lambda s: s.get("coinflip_money_won", 0) >= 1000000)

# Coinflip - Lost
Achievement("coinflip_lost_1k", "🌧️", 10,
            lambda s: s.get("coinflip_money_lost", 0) >= 1000)
Achievement("coinflip_lost_5k", "🌧️", 20,
            lambda s: s.get("coinflip_money_lost", 0) >= 5000)
Achievement("coinflip_lost_100k", "💸", 100,
            lambda s: s.get("coinflip_money_lost", 0) >= 100000)
Achievement("coinflip_lost_1m", "🤡", 500,
            lambda s: s.get("coinflip_money_lost", 0) >= 1000000)

# Slots - Won
Achievement("slots_won_1k", "🎰", 10,
            lambda s: s.get("slots_money_won", 0) >= 1000)
Achievement("slots_won_5k", "🎰", 20,
            lambda s: s.get("slots_money_won", 0) >= 5000)
Achievement("slots_won_100k", "🎰", 100,
            lambda s: s.get("slots_money_won", 0) >= 100000)
Achievement("slots_won_1m", "🎰", 500,
            lambda s: s.get("slots_money_won", 0) >= 1000000)

# Slots - Lost
Achievement("slots_lost_1k", "😠", 10,
            lambda s: s.get("slots_money_lost", 0) >= 1000)
Achievement("slots_lost_5k", "😠", 20,
            lambda s: s.get("slots_money_lost", 0) >= 5000)
Achievement("slots_lost_100k", "📉", 100,
            lambda s: s.get("slots_money_lost", 0) >= 100000)
Achievement("slots_lost_1m", "💸", 500,
            lambda s: s.get("slots_money_lost", 0) >= 1000000)

# Blackjack - Won
Achievement("blackjack_won_1k", "🃏", 10,
            lambda s: s.get("blackjack_money_won", 0) >= 1000)
Achievement("blackjack_won_5k", "🃏", 20,
            lambda s: s.get("blackjack_money_won", 0) >= 5000)
Achievement("blackjack_won_100k", "🃏", 100,
            lambda s: s.get("blackjack_money_won", 0) >= 100000)
Achievement("blackjack_won_1m", "🃏", 500,
            lambda s: s.get("blackjack_money_won", 0) >= 1000000)

# Blackjack - Lost
Achievement("blackjack_lost_1k", "🤦", 10,
            lambda s: s.get("blackjack_money_lost", 0) >= 1000)
Achievement("blackjack_lost_5k", "🤦", 20,
            lambda s: s.get("blackjack_money_lost", 0) >= 5000)
Achievement("blackjack_lost_100k", "😔", 100,
            lambda s: s.get("blackjack_money_lost", 0) >= 100000)
Achievement("blackjack_lost_1m", "🏚️", 500,
            lambda s: s.get("blackjack_money_lost", 0) >= 1000000)

# Roulette - Won
Achievement("roulette_won_1k", "🔫", 10,
            lambda s: s.get("roulette_money_won", 0) >= 1000)
Achievement("roulette_won_5k", "🔫", 20,
            lambda s: s.get("roulette_money_won", 0) >= 5000)
Achievement("roulette_won_100k", "🔫", 100,
            lambda s: s.get("roulette_money_won", 0) >= 100000)
Achievement("roulette_won_1m", "🔫", 500,
            lambda s: s.get("roulette_money_won", 0) >= 1000000)

# Roulette - Lost
Achievement("roulette_lost_1k", "🩸", 10,
            lambda s: s.get("roulette_money_lost", 0) >= 1000)
Achievement("roulette_lost_5k", "🩸", 20,
            lambda s: s.get("roulette_money_lost", 0) >= 5000)
Achievement("roulette_lost_100k", "💀", 100,
            lambda s: s.get("roulette_money_lost", 0) >= 100000)
Achievement("roulette_lost_1m", "🪦", 500,
            lambda s: s.get("roulette_money_lost", 0) >= 1000000)

# --- BETTING ---
Achievement("bet_rookie", "🎲", 10,
            lambda s: s.get("wagers_won", 0) >= 1)

Achievement("bet_pro", "🔮", 50,
            lambda s: s.get("wagers_won", 0) >= 25)



# --- LOTTO ---
Achievement("lotto_rookie", "🎫", 10,
            lambda s: s.get("lotto_participations", 0) >= 10)

Achievement("lotto_winner", "🎉", 200,
            lambda s: s.get("lotto_won", 0) >= 1)

# --- ECONOMY / DAILY ---
Achievement("daily_1", "📅", 10,
            lambda s: s.get("daily_uses", 0) >= 1)

Achievement("daily_10", "📅", 20,
            lambda s: s.get("daily_uses", 0) >= 10)
            
Achievement("daily_50", "📅", 50,
            lambda s: s.get("daily_uses", 0) >= 50)
            
Achievement("daily_100", "📅", 100,
            lambda s: s.get("daily_uses", 0) >= 100)

Achievement("daily_365", "📅", 500,
            lambda s: s.get("daily_uses", 0) >= 365)


# --- PET COLLECTION ---
Achievement("pet_collector_common", "🦋", 50,
            lambda s: s.get("collected_common_pets", 0) >= 8)

Achievement("pet_collector_rare", "🦁", 150,
            lambda s: s.get("collected_rare_pets", 0) >= 6)

Achievement("pet_collector_epic", "🦄", 300,
            lambda s: s.get("collected_epic_pets", 0) >= 4)

Achievement("pet_collector_legendary", "🐉", 500,
            lambda s: s.get("collected_legendary_pets", 0) >= 1)

Achievement("pet_collector_all", "🌍", 1000,
            lambda s: s.get("collected_all_pets", 0) >= 19)

# --- PET RANKS ---
Achievement("rank_bronze", "🥉", 50,
            lambda s: any("Bronze" in r for r in s.get("pet_ranks", [])))

Achievement("rank_silver", "🥈", 100,
            lambda s: any("Argent" in r for r in s.get("pet_ranks", [])))

Achievement("rank_gold", "🥇", 500,
            lambda s: any("Or" in r for r in s.get("pet_ranks", [])))

Achievement("rank_diamond", "💎", 1000,
            lambda s: any("Diamant" in r for r in s.get("pet_ranks", [])))

Achievement("rank_top5", "🌟", 5000,
            lambda s: any("Top 5" in r for r in s.get("pet_ranks", [])))

# --- COMMUNITY ---
Achievement("community_initiate", "🧱", 10,
            lambda s: s.get("community_money", 0) >= 10000 or s.get("community_items", 0) >= 200)

Achievement("community_supporter", "🏛️", 50,
            lambda s: s.get("community_money", 0) >= 500000 or s.get("community_items", 0) >= 5000)

Achievement("community_pillar", "🏛️", 150,
            lambda s: s.get("community_money", 0) >= 5000000 or s.get("community_items", 0) >= 50000)

# --- BOSS LEAGUE ---
Achievement("boss_league_1", "⚔️", 20,
            lambda s: s.get("boss_league_stage", 0) >= 1)

Achievement("boss_league_2", "🏹", 50,
            lambda s: s.get("boss_league_stage", 0) >= 2)

Achievement("boss_league_3", "🛡️", 100,
            lambda s: s.get("boss_league_stage", 0) >= 3)

Achievement("boss_league_4", "⚡", 200,
            lambda s: s.get("boss_league_stage", 0) >= 4)

Achievement("boss_league_5", "🏆", 500,
            lambda s: s.get("boss_league_stage", 0) >= 5)